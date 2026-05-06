package service

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbgroup "github.com/Wei-Shaw/sub2api/ent/group"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// AssignProxiesToAccountsInput describes a guarded proxy assignment request.
type AssignProxiesToAccountsInput struct {
	ProxyIDs []int64
	DryRun   bool
	Filters  ProxyAssignmentAccountFilters
}

// ProxyAssignmentAccountFilters limits the target account set.
type ProxyAssignmentAccountFilters struct {
	Platforms []string
	GroupIDs  []int64
	Statuses  []string
}

type ProxyAccountAssignmentResult struct {
	DryRun                 bool                          `json:"dry_run"`
	MatchedAccountCount    int                           `json:"matched_account_count"`
	UniqueAccountCount     int                           `json:"unique_account_count"`
	DuplicateHitCount      int                           `json:"duplicate_hit_count"`
	PlannedAssignmentCount int                           `json:"planned_assignment_count"`
	ActualAssignmentCount  int                           `json:"actual_assignment_count"`
	Proxies                []ProxyAccountAssignmentProxy `json:"proxies"`
}

type ProxyAccountAssignmentProxy struct {
	ProxyID            int64                           `json:"proxy_id"`
	ProxyName          string                          `json:"proxy_name"`
	BeforeAccountCount int64                           `json:"before_account_count"`
	PlannedCount       int                             `json:"planned_count"`
	AssignedCount      int                             `json:"assigned_count"`
	AfterAccountCount  int64                           `json:"after_account_count"`
	Accounts           []ProxyAccountAssignmentAccount `json:"accounts"`
}

type ProxyAccountAssignmentAccount struct {
	AccountID     int64  `json:"account_id"`
	AccountName   string `json:"account_name"`
	Platform      string `json:"platform"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	Assigned      bool   `json:"assigned"`
	SkippedReason string `json:"skipped_reason,omitempty"`
}

type proxyAssignmentTargetAccount struct {
	ID       int64
	Name     string
	Platform string
	Type     string
	Status   string
	GroupIDs []int64
}

type accountProxyAssigner interface {
	AssignProxyIDsIfUnassigned(ctx context.Context, assignments map[int64]int64) (map[int64]bool, error)
}

func (s *adminServiceImpl) AssignProxiesToAccounts(
	ctx context.Context,
	input *AssignProxiesToAccountsInput,
) (*ProxyAccountAssignmentResult, error) {
	if input == nil {
		return nil, infraerrors.BadRequest("INVALID_INPUT", "assignment input is required")
	}

	proxyIDs := dedupePositiveInt64s(input.ProxyIDs)
	if len(proxyIDs) == 0 {
		return nil, infraerrors.BadRequest("NO_PROXY_SELECTED", "at least one proxy must be selected")
	}

	proxies, err := s.loadSelectedAssignmentProxies(ctx, proxyIDs)
	if err != nil {
		return nil, err
	}

	counts, err := s.loadSelectedProxyAccountCounts(ctx, proxies)
	if err != nil {
		return nil, err
	}

	targets, matched, duplicateHits, err := s.listProxyAssignmentTargets(ctx, input.Filters)
	if err != nil {
		return nil, err
	}

	result := buildProxyAssignmentPlan(input.DryRun, proxies, counts, targets)
	result.MatchedAccountCount = matched
	result.UniqueAccountCount = len(targets)
	result.DuplicateHitCount = duplicateHits

	if input.DryRun || result.PlannedAssignmentCount == 0 {
		return result, nil
	}

	applied, err := s.applyProxyAssignments(ctx, result)
	if err != nil {
		return nil, err
	}
	markAppliedProxyAssignments(result, applied)
	return result, nil
}

func (s *adminServiceImpl) loadSelectedAssignmentProxies(ctx context.Context, ids []int64) ([]Proxy, error) {
	if s.proxyRepo == nil {
		return nil, infraerrors.InternalServer("PROXY_REPOSITORY_UNAVAILABLE", "proxy repository is unavailable")
	}
	proxies, err := s.proxyRepo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]Proxy, len(proxies))
	for _, proxy := range proxies {
		byID[proxy.ID] = proxy
	}
	if len(byID) != len(ids) {
		return nil, infraerrors.BadRequest("PROXY_NOT_FOUND", "one or more selected proxies do not exist")
	}
	ordered := make([]Proxy, 0, len(ids))
	for _, id := range ids {
		ordered = append(ordered, byID[id])
	}
	return ordered, nil
}

func (s *adminServiceImpl) loadSelectedProxyAccountCounts(
	ctx context.Context,
	proxies []Proxy,
) (map[int64]int64, error) {
	counts := make(map[int64]int64, len(proxies))
	if s.proxyRepo == nil {
		return nil, infraerrors.InternalServer("PROXY_REPOSITORY_UNAVAILABLE", "proxy repository is unavailable")
	}
	for _, proxy := range proxies {
		count, err := s.proxyRepo.CountAccountsByProxyID(ctx, proxy.ID)
		if err != nil {
			return nil, err
		}
		counts[proxy.ID] = count
	}
	return counts, nil
}

func (s *adminServiceImpl) listProxyAssignmentTargets(
	ctx context.Context,
	filters ProxyAssignmentAccountFilters,
) ([]proxyAssignmentTargetAccount, int, int, error) {
	if s.entClient == nil {
		return nil, 0, 0, infraerrors.InternalServer("ENT_CLIENT_UNAVAILABLE", "database client is unavailable")
	}

	groupIDs := dedupePositiveInt64s(filters.GroupIDs)
	q := s.entClient.Account.Query().
		Where(dbaccount.DeletedAtIsNil(), dbaccount.ProxyIDIsNil()).
		WithGroups().
		Order(dbaccount.ByID())

	if platforms := normalizeStringFilters(filters.Platforms); len(platforms) > 0 {
		q = q.Where(dbaccount.PlatformIn(platforms...))
	}
	if statuses := normalizeAccountStatusFilters(filters.Statuses); len(statuses) > 0 {
		q = q.Where(dbaccount.StatusIn(statuses...))
	}
	if len(groupIDs) > 0 {
		q = q.Where(dbaccount.HasGroupsWith(dbgroup.IDIn(groupIDs...)))
	}

	entities, err := q.All(ctx)
	if err != nil {
		return nil, 0, 0, err
	}

	targets, matchedRows := assignmentTargetsFromEntities(entities, groupIDs)
	duplicateHits := matchedRows - len(targets)
	if duplicateHits < 0 {
		duplicateHits = 0
	}
	return targets, matchedRows, duplicateHits, nil
}

func assignmentTargetsFromEntities(
	entities []*dbent.Account,
	groupIDs []int64,
) ([]proxyAssignmentTargetAccount, int) {
	targets := make([]proxyAssignmentTargetAccount, 0, len(entities))
	matchedRows := 0
	selectedGroups := int64Set(groupIDs)
	for _, entity := range entities {
		groupIDsForAccount := make([]int64, 0, len(entity.Edges.Groups))
		groupMatches := 0
		for _, group := range entity.Edges.Groups {
			if group == nil {
				continue
			}
			groupIDsForAccount = append(groupIDsForAccount, group.ID)
			if len(selectedGroups) > 0 {
				if _, ok := selectedGroups[group.ID]; ok {
					groupMatches++
				}
			}
		}
		if len(selectedGroups) == 0 {
			groupMatches = 1
		}
		matchedRows += groupMatches
		targets = append(targets, proxyAssignmentTargetAccount{
			ID:       entity.ID,
			Name:     entity.Name,
			Platform: entity.Platform,
			Type:     entity.Type,
			Status:   entity.Status,
			GroupIDs: groupIDsForAccount,
		})
	}

	return targets, matchedRows
}

func buildProxyAssignmentPlan(
	dryRun bool,
	proxies []Proxy,
	counts map[int64]int64,
	targets []proxyAssignmentTargetAccount,
) *ProxyAccountAssignmentResult {
	rows := make([]ProxyAccountAssignmentProxy, 0, len(proxies))
	for _, proxy := range proxies {
		before := counts[proxy.ID]
		rows = append(rows, ProxyAccountAssignmentProxy{
			ProxyID:            proxy.ID,
			ProxyName:          proxy.Name,
			BeforeAccountCount: before,
			AfterAccountCount:  before,
			Accounts:           []ProxyAccountAssignmentAccount{},
		})
	}

	result := &ProxyAccountAssignmentResult{DryRun: dryRun, Proxies: rows}
	if len(result.Proxies) == 0 {
		return result
	}

	for _, target := range targets {
		idx := lowestLoadProxyIndex(result.Proxies)
		account := ProxyAccountAssignmentAccount{
			AccountID:   target.ID,
			AccountName: target.Name,
			Platform:    target.Platform,
			Type:        target.Type,
			Status:      target.Status,
		}
		result.Proxies[idx].Accounts = append(result.Proxies[idx].Accounts, account)
		result.Proxies[idx].PlannedCount++
		result.Proxies[idx].AfterAccountCount++
		result.PlannedAssignmentCount++
	}
	return result
}

func lowestLoadProxyIndex(proxies []ProxyAccountAssignmentProxy) int {
	best := 0
	for i := 1; i < len(proxies); i++ {
		if proxies[i].AfterAccountCount < proxies[best].AfterAccountCount {
			best = i
			continue
		}
		if proxies[i].AfterAccountCount == proxies[best].AfterAccountCount &&
			proxies[i].ProxyID < proxies[best].ProxyID {
			best = i
		}
	}
	return best
}

func (s *adminServiceImpl) applyProxyAssignments(
	ctx context.Context,
	result *ProxyAccountAssignmentResult,
) (map[int64]bool, error) {
	assigner, ok := s.accountRepo.(accountProxyAssigner)
	if !ok || assigner == nil {
		return nil, infraerrors.InternalServer("ACCOUNT_ASSIGNMENT_UNAVAILABLE", "account assignment repository is unavailable")
	}

	assignments := make(map[int64]int64, result.PlannedAssignmentCount)
	for _, proxy := range result.Proxies {
		for _, account := range proxy.Accounts {
			assignments[account.AccountID] = proxy.ProxyID
		}
	}
	return assigner.AssignProxyIDsIfUnassigned(ctx, assignments)
}

func markAppliedProxyAssignments(result *ProxyAccountAssignmentResult, applied map[int64]bool) {
	result.ActualAssignmentCount = 0
	for pIdx := range result.Proxies {
		proxy := &result.Proxies[pIdx]
		proxy.AssignedCount = 0
		proxy.AfterAccountCount = proxy.BeforeAccountCount
		for aIdx := range proxy.Accounts {
			account := &proxy.Accounts[aIdx]
			if applied[account.AccountID] {
				account.Assigned = true
				proxy.AssignedCount++
				proxy.AfterAccountCount++
				result.ActualAssignmentCount++
				continue
			}
			account.SkippedReason = "account already has a proxy"
		}
	}
}

func int64Set(values []int64) map[int64]struct{} {
	out := make(map[int64]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func assignmentRange(counts []int64) int64 {
	if len(counts) == 0 {
		return 0
	}
	minValue := counts[0]
	maxValue := counts[0]
	for _, count := range counts[1:] {
		if count < minValue {
			minValue = count
		}
		if count > maxValue {
			maxValue = count
		}
	}
	return maxValue - minValue
}
