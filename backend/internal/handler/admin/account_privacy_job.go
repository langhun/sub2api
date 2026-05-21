package admin

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

func (h *AccountHandler) getBatchPrivacyJobRegistry() *batchPrivacyJobRegistry {
	if h.privacyJobs != nil {
		return h.privacyJobs
	}
	h.privacyJobs = newBatchPrivacyJobRegistry()
	return h.privacyJobs
}

func (h *AccountHandler) enqueueBatchPrivacyJob(
	c *gin.Context,
	op batchPrivacyOperation,
	requestedIDs []int64,
) {
	accountIDs, ok := prepareBatchPrivacyRequest(c, requestedIDs)
	if !ok {
		return
	}

	job := h.getBatchPrivacyJobRegistry().create(op, len(requestedIDs), len(accountIDs))
	go h.runBatchPrivacyJob(job.JobID, op, accountIDs)

	response.Success(c, gin.H{
		"job_id":             job.JobID,
		"operation":          job.Operation,
		"status":             job.Status,
		"requested_total":    job.RequestedTotal,
		"deduplicated_total": job.DeduplicatedTotal,
		"duplicates_removed": job.DuplicatesRemoved,
	})
}

func (h *AccountHandler) runBatchPrivacyJob(
	jobID string,
	op batchPrivacyOperation,
	accountIDs []int64,
) {
	registry := h.getBatchPrivacyJobRegistry()
	ctx, cancel := context.WithTimeout(context.Background(), batchPrivacyAsyncTimeout)
	defer cancel()

	select {
	case registry.workerSem <- struct{}{}:
		defer func() { <-registry.workerSem }()
	case <-ctx.Done():
		registry.markFinished(jobID, batchPrivacyJobFailed, nil, ctx.Err().Error())
		return
	}

	registry.markRunning(jobID)
	defer func() {
		if r := recover(); r != nil {
			registry.markFinished(jobID, batchPrivacyJobFailed, nil, fmt.Sprintf("panic: %v", r))
		}
	}()

	result, err := h.executeBatchPrivacyOperation(ctx, op, accountIDs)
	if err != nil {
		registry.markFinished(jobID, batchPrivacyJobFailed, &result, err.Error())
		return
	}
	registry.markFinished(jobID, batchPrivacyJobCompleted, &result, "")
}

func (h *AccountHandler) executeBatchPrivacyOperation(
	ctx context.Context,
	op batchPrivacyOperation,
	accountIDs []int64,
) (batchPrivacyJobResult, error) {
	result := batchPrivacyJobResult{
		Total:  len(accountIDs),
		Errors: make([]batchPrivacyJobError, 0),
	}
	if h.adminService == nil {
		return result, fmt.Errorf("admin service is not configured")
	}

	accounts, err := h.adminService.GetAccountsByIDs(ctx, accountIDs)
	if err != nil {
		return result, err
	}

	foundIDs := make(map[int64]bool, len(accounts))
	for _, acc := range accounts {
		if acc != nil {
			foundIDs[acc.ID] = true
		}
	}
	for _, id := range accountIDs {
		if !foundIDs[id] {
			result.Failed++
			result.Errors = append(result.Errors, batchPrivacyJobError{
				AccountID: id,
				Error:     "account not found",
			})
		}
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(batchPrivacyConcurrency(op))

	var mu sync.Mutex
	for _, account := range accounts {
		acc := account
		if acc == nil {
			continue
		}
		g.Go(func() error {
			update := h.executeBatchPrivacyAccount(gctx, op, acc)
			mu.Lock()
			mergeBatchPrivacyResult(&result, update)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return result, err
	}
	return result, nil
}

func batchPrivacyConcurrency(op batchPrivacyOperation) int {
	if op == batchPrivacyOperationClear {
		return 10
	}
	return 5
}

func (h *AccountHandler) executeBatchPrivacyAccount(
	ctx context.Context,
	op batchPrivacyOperation,
	account *service.Account,
) batchPrivacyJobResult {
	accountCtx, cancel := context.WithTimeout(ctx, batchPrivacyPerAccountTTL)
	defer cancel()

	result := batchPrivacyJobResult{}
	if account.Platform != service.PlatformOpenAI || account.Type != service.AccountTypeOAuth {
		result.Skipped = 1
		return result
	}

	switch op {
	case batchPrivacyOperationSet:
		return h.executeBatchSetPrivacyAccount(accountCtx, account)
	case batchPrivacyOperationClear:
		return h.executeBatchClearPrivacyAccount(accountCtx, account)
	default:
		result.Failed = 1
		result.Errors = []batchPrivacyJobError{{
			AccountID: account.ID,
			Error:     fmt.Sprintf("unsupported privacy operation %q", op),
		}}
		return result
	}
}

func (h *AccountHandler) executeBatchSetPrivacyAccount(
	ctx context.Context,
	account *service.Account,
) batchPrivacyJobResult {
	mode := h.adminService.ForceOpenAIPrivacy(ctx, account)
	switch mode {
	case service.PrivacyModeTrainingOff:
		return batchPrivacyJobResult{Success: 1}
	case service.PrivacyModeCFBlocked:
		return failedBatchPrivacyAccount(
			account.ID,
			"failed to set privacy: upstream request blocked by Cloudflare",
		)
	case "", service.PrivacyModeFailed:
		return failedBatchPrivacyAccount(
			account.ID,
			"failed to set privacy: missing access_token or upstream request failed",
		)
	default:
		return failedBatchPrivacyAccount(
			account.ID,
			fmt.Sprintf("failed to set privacy: unexpected privacy mode %q", mode),
		)
	}
}

func (h *AccountHandler) executeBatchClearPrivacyAccount(
	ctx context.Context,
	account *service.Account,
) batchPrivacyJobResult {
	if err := h.adminService.ClearAccountPrivacyMode(ctx, account); err != nil {
		return failedBatchPrivacyAccount(account.ID, err.Error())
	}
	return batchPrivacyJobResult{Success: 1}
}

func failedBatchPrivacyAccount(accountID int64, message string) batchPrivacyJobResult {
	return batchPrivacyJobResult{
		Failed: 1,
		Errors: []batchPrivacyJobError{{
			AccountID: accountID,
			Error:     message,
		}},
	}
}

func mergeBatchPrivacyResult(dst *batchPrivacyJobResult, src batchPrivacyJobResult) {
	dst.Success += src.Success
	dst.Failed += src.Failed
	dst.Skipped += src.Skipped
	if len(src.Errors) > 0 {
		dst.Errors = append(dst.Errors, src.Errors...)
	}
}

// GetBatchPrivacyJob returns status for background OpenAI privacy batch jobs.
// GET /api/v1/admin/accounts/batch-privacy-jobs/:job_id
func (h *AccountHandler) GetBatchPrivacyJob(c *gin.Context) {
	jobID := strings.TrimSpace(c.Param("job_id"))
	if jobID == "" {
		response.BadRequest(c, "Invalid privacy job ID")
		return
	}

	job, ok := h.getBatchPrivacyJobRegistry().get(jobID)
	if !ok {
		response.NotFound(c, "Privacy job not found")
		return
	}
	response.Success(c, job)
}

func prepareBatchPrivacyRequest(c *gin.Context, requestedIDs []int64) ([]int64, bool) {
	if len(requestedIDs) > batchPrivacyMaxRequestIDs {
		response.BadRequest(c, fmt.Sprintf("account_ids exceeds maximum of %d", batchPrivacyMaxRequestIDs))
		return nil, false
	}

	accountIDs := normalizeInt64IDList(requestedIDs)
	if len(accountIDs) == 0 {
		response.BadRequest(c, "account_ids must contain at least one valid id")
		return nil, false
	}
	return accountIDs, true
}
