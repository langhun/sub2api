package admin

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	batchPrivacyAsyncThreshold = 20
	batchPrivacyMaxRequestIDs  = 2000
	batchPrivacyMaxStoredJobs  = 200
	batchPrivacyConcurrentJobs = 2
	batchPrivacySyncTimeout    = 30 * time.Second
	batchPrivacyPerAccountTTL  = 15 * time.Second
	batchPrivacyAsyncTimeout   = 10 * time.Minute
	batchPrivacyJobRetention   = 24 * time.Hour
)

type batchPrivacyOperation string

const (
	batchPrivacyOperationSet   batchPrivacyOperation = "set"
	batchPrivacyOperationClear batchPrivacyOperation = "clear"
)

type batchPrivacyJobStatus string

const (
	batchPrivacyJobQueued    batchPrivacyJobStatus = "queued"
	batchPrivacyJobRunning   batchPrivacyJobStatus = "running"
	batchPrivacyJobCompleted batchPrivacyJobStatus = "completed"
	batchPrivacyJobFailed    batchPrivacyJobStatus = "failed"
)

type batchPrivacyJobError struct {
	AccountID int64  `json:"account_id"`
	Error     string `json:"error"`
}

type batchPrivacyJobResult struct {
	Total   int                    `json:"total"`
	Success int                    `json:"success"`
	Failed  int                    `json:"failed"`
	Skipped int                    `json:"skipped"`
	Errors  []batchPrivacyJobError `json:"errors"`
}

type batchPrivacyJob struct {
	JobID             string                 `json:"job_id"`
	Operation         batchPrivacyOperation  `json:"operation"`
	Status            batchPrivacyJobStatus  `json:"status"`
	RequestedTotal    int                    `json:"requested_total"`
	DeduplicatedTotal int                    `json:"deduplicated_total"`
	DuplicatesRemoved int                    `json:"duplicates_removed"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	StartedAt         *time.Time             `json:"started_at,omitempty"`
	FinishedAt        *time.Time             `json:"finished_at,omitempty"`
	Result            *batchPrivacyJobResult `json:"result,omitempty"`
	Error             string                 `json:"error,omitempty"`
}

type batchPrivacyJobRegistry struct {
	mu        sync.RWMutex
	seq       uint64
	jobs      map[string]*batchPrivacyJob
	workerSem chan struct{}
}

func newBatchPrivacyJobRegistry() *batchPrivacyJobRegistry {
	return &batchPrivacyJobRegistry{
		jobs:      make(map[string]*batchPrivacyJob),
		workerSem: make(chan struct{}, batchPrivacyConcurrentJobs),
	}
}

func (r *batchPrivacyJobRegistry) create(
	op batchPrivacyOperation,
	requestedTotal int,
	deduplicatedTotal int,
) *batchPrivacyJob {
	now := time.Now().UTC()
	seq := atomic.AddUint64(&r.seq, 1)
	job := &batchPrivacyJob{
		JobID:             fmt.Sprintf("privacy-%d-%d", now.UnixNano(), seq),
		Operation:         op,
		Status:            batchPrivacyJobQueued,
		RequestedTotal:    requestedTotal,
		DeduplicatedTotal: deduplicatedTotal,
		DuplicatesRemoved: requestedTotal - deduplicatedTotal,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	r.mu.Lock()
	r.pruneLocked(now)
	r.jobs[job.JobID] = job
	r.mu.Unlock()
	return cloneBatchPrivacyJob(job)
}

func (r *batchPrivacyJobRegistry) pruneLocked(now time.Time) {
	if len(r.jobs) <= batchPrivacyMaxStoredJobs {
		return
	}
	cutoff := now.Add(-batchPrivacyJobRetention)
	for id, job := range r.jobs {
		if job == nil || job.FinishedAt == nil || job.FinishedAt.After(cutoff) {
			continue
		}
		delete(r.jobs, id)
	}
}

func (r *batchPrivacyJobRegistry) markRunning(jobID string) {
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()

	job := r.jobs[jobID]
	if job == nil {
		return
	}
	job.Status = batchPrivacyJobRunning
	job.UpdatedAt = now
	job.StartedAt = &now
}

func (r *batchPrivacyJobRegistry) markFinished(
	jobID string,
	status batchPrivacyJobStatus,
	result *batchPrivacyJobResult,
	errorMessage string,
) {
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()

	job := r.jobs[jobID]
	if job == nil {
		return
	}
	job.Status = status
	job.UpdatedAt = now
	job.FinishedAt = &now
	job.Result = cloneBatchPrivacyJobResult(result)
	job.Error = errorMessage
}

func (r *batchPrivacyJobRegistry) get(jobID string) (*batchPrivacyJob, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job := r.jobs[jobID]
	if job == nil {
		return nil, false
	}
	return cloneBatchPrivacyJob(job), true
}

func cloneBatchPrivacyJob(job *batchPrivacyJob) *batchPrivacyJob {
	if job == nil {
		return nil
	}

	out := *job
	if job.StartedAt != nil {
		startedAt := *job.StartedAt
		out.StartedAt = &startedAt
	}
	if job.FinishedAt != nil {
		finishedAt := *job.FinishedAt
		out.FinishedAt = &finishedAt
	}
	out.Result = cloneBatchPrivacyJobResult(job.Result)
	return &out
}

func cloneBatchPrivacyJobResult(result *batchPrivacyJobResult) *batchPrivacyJobResult {
	if result == nil {
		return nil
	}
	out := *result
	out.Errors = append([]batchPrivacyJobError(nil), result.Errors...)
	return &out
}
