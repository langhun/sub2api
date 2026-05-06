package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbgroup "github.com/Wei-Shaw/sub2api/ent/group"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	AccountDuplicateSeverityStrong = "strong"
	AccountDuplicateSeverityWeak   = "weak"
)

var strongDuplicateCredentialKeys = []string{
	"chatgpt_account_id",
	"chatgpt_user_id",
	"account_uuid",
	"org_uuid",
	"project_id",
	"api_key",
	"refresh_token",
}

// AccountDuplicateCheckInput describes a duplicate-account scan.
type AccountDuplicateCheckInput struct {
	Platforms       []string
	GroupIDs        []int64
	Statuses        []string
	IncludeInactive *bool
}

type AccountDuplicateCheckResult struct {
	TotalAccounts         int                     `json:"total_accounts"`
	DuplicateGroupCount   int                     `json:"duplicate_group_count"`
	DuplicateAccountCount int                     `json:"duplicate_account_count"`
	StrongGroupCount      int                     `json:"strong_group_count"`
	WeakGroupCount        int                     `json:"weak_group_count"`
	Groups                []AccountDuplicateGroup `json:"groups"`
}

type AccountDuplicateGroup struct {
	KeyType      string                    `json:"key_type"`
	Severity     string                    `json:"severity"`
	Platform     string                    `json:"platform"`
	Type         string                    `json:"type,omitempty"`
	MaskedValue  string                    `json:"masked_value"`
	ValueHash    string                    `json:"value_hash"`
	AccountCount int                       `json:"account_count"`
	Accounts     []AccountDuplicateAccount `json:"accounts"`
}

type AccountDuplicateAccount struct {
	ID       int64                          `json:"id"`
	Name     string                         `json:"name"`
	Platform string                         `json:"platform"`
	Type     string                         `json:"type"`
	Status   string                         `json:"status"`
	ProxyID  *int64                         `json:"proxy_id"`
	Groups   []AccountDuplicateAccountGroup `json:"groups"`
}

type AccountDuplicateAccountGroup struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type duplicateCheckAccount struct {
	ID          int64
	Name        string
	Platform    string
	Type        string
	Status      string
	ProxyID     *int64
	Credentials map[string]any
	Extra       map[string]any
	Groups      []AccountDuplicateAccountGroup
}

type duplicateBucket struct {
	keyType  string
	severity string
	platform string
	accType  string
	rawValue string
	accounts []duplicateCheckAccount
}

func (s *adminServiceImpl) CheckDuplicateAccounts(
	ctx context.Context,
	input *AccountDuplicateCheckInput,
) (*AccountDuplicateCheckResult, error) {
	if s.entClient == nil {
		return nil, infraerrors.InternalServer("ENT_CLIENT_UNAVAILABLE", "database client is unavailable")
	}
	if input == nil {
		input = &AccountDuplicateCheckInput{}
	}

	accounts, err := s.listDuplicateCheckAccounts(ctx, input)
	if err != nil {
		return nil, err
	}
	return buildAccountDuplicateCheckResult(accounts), nil
}

func (s *adminServiceImpl) listDuplicateCheckAccounts(
	ctx context.Context,
	input *AccountDuplicateCheckInput,
) ([]duplicateCheckAccount, error) {
	groupIDs := dedupePositiveInt64s(input.GroupIDs)
	q := s.entClient.Account.Query().
		Where(dbaccount.DeletedAtIsNil()).
		WithGroups().
		Order(dbaccount.ByID())

	if platforms := normalizeStringFilters(input.Platforms); len(platforms) > 0 {
		q = q.Where(dbaccount.PlatformIn(platforms...))
	}
	if statuses := normalizeAccountStatusFilters(input.Statuses); len(statuses) > 0 {
		q = q.Where(dbaccount.StatusIn(statuses...))
	} else if input.IncludeInactive != nil && !*input.IncludeInactive {
		q = q.Where(dbaccount.StatusEQ(StatusActive))
	}
	if len(groupIDs) > 0 {
		q = q.Where(dbaccount.HasGroupsWith(dbgroup.IDIn(groupIDs...)))
	}

	entities, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	accounts := make([]duplicateCheckAccount, 0, len(entities))
	for _, entity := range entities {
		groups := make([]AccountDuplicateAccountGroup, 0, len(entity.Edges.Groups))
		for _, group := range entity.Edges.Groups {
			if group == nil {
				continue
			}
			groups = append(groups, AccountDuplicateAccountGroup{ID: group.ID, Name: group.Name})
		}
		accounts = append(accounts, duplicateCheckAccount{
			ID:          entity.ID,
			Name:        entity.Name,
			Platform:    entity.Platform,
			Type:        entity.Type,
			Status:      entity.Status,
			ProxyID:     entity.ProxyID,
			Credentials: copyAnyMap(entity.Credentials),
			Extra:       copyAnyMap(entity.Extra),
			Groups:      groups,
		})
	}
	return accounts, nil
}

func buildAccountDuplicateCheckResult(accounts []duplicateCheckAccount) *AccountDuplicateCheckResult {
	strongGroups := collectStrongDuplicateGroups(accounts)
	strongAccountIDs := collectDuplicateAccountIDs(strongGroups)
	weakGroups := collectWeakDuplicateGroups(accounts, strongAccountIDs)
	groups := append(strongGroups, weakGroups...)
	sortDuplicateGroups(groups)

	result := &AccountDuplicateCheckResult{
		TotalAccounts:         len(accounts),
		DuplicateGroupCount:   len(groups),
		DuplicateAccountCount: len(collectDuplicateAccountIDs(groups)),
		Groups:                groups,
	}
	for _, group := range groups {
		if group.Severity == AccountDuplicateSeverityStrong {
			result.StrongGroupCount++
		} else {
			result.WeakGroupCount++
		}
	}
	return result
}

func collectStrongDuplicateGroups(accounts []duplicateCheckAccount) []AccountDuplicateGroup {
	buckets := make(map[string]*duplicateBucket)
	for _, account := range accounts {
		for _, key := range strongDuplicateCredentialKeys {
			rawValue := accountFieldString(account.Credentials, key)
			if rawValue == "" {
				continue
			}
			normalized := normalizeStrongDuplicateValue(key, rawValue)
			bucketKey := strings.Join([]string{account.Platform, account.Type, key, normalized}, "\x00")
			addDuplicateBucketAccount(buckets, bucketKey, duplicateBucket{
				keyType:  key,
				severity: AccountDuplicateSeverityStrong,
				platform: account.Platform,
				accType:  account.Type,
				rawValue: rawValue,
			}, account)
		}
	}
	return duplicateBucketsToGroups(buckets)
}

func collectWeakDuplicateGroups(
	accounts []duplicateCheckAccount,
	strongAccountIDs map[int64]struct{},
) []AccountDuplicateGroup {
	buckets := make(map[string]*duplicateBucket)
	for _, account := range accounts {
		if _, strong := strongAccountIDs[account.ID]; strong {
			continue
		}
		addWeakDuplicateCandidate(buckets, account, "email_address", accountFieldString(account.Extra, "email_address"))
		addWeakDuplicateCandidate(buckets, account, "name", account.Name)
	}
	return duplicateBucketsToGroups(buckets)
}

func addWeakDuplicateCandidate(
	buckets map[string]*duplicateBucket,
	account duplicateCheckAccount,
	keyType string,
	rawValue string,
) {
	normalized := normalizeWeakDuplicateValue(rawValue)
	if normalized == "" {
		return
	}
	bucketKey := strings.Join([]string{account.Platform, keyType, normalized}, "\x00")
	addDuplicateBucketAccount(buckets, bucketKey, duplicateBucket{
		keyType:  keyType,
		severity: AccountDuplicateSeverityWeak,
		platform: account.Platform,
		rawValue: rawValue,
	}, account)
}

func addDuplicateBucketAccount(
	buckets map[string]*duplicateBucket,
	key string,
	seed duplicateBucket,
	account duplicateCheckAccount,
) {
	bucket := buckets[key]
	if bucket == nil {
		clone := seed
		bucket = &clone
		buckets[key] = bucket
	}
	bucket.accounts = append(bucket.accounts, account)
}

func duplicateBucketsToGroups(buckets map[string]*duplicateBucket) []AccountDuplicateGroup {
	groups := make([]AccountDuplicateGroup, 0, len(buckets))
	for _, bucket := range buckets {
		if len(bucket.accounts) < 2 {
			continue
		}
		group := AccountDuplicateGroup{
			KeyType:      bucket.keyType,
			Severity:     bucket.severity,
			Platform:     bucket.platform,
			Type:         bucket.accType,
			MaskedValue:  maskDuplicateValue(bucket.rawValue),
			ValueHash:    shortSHA256(bucket.rawValue),
			AccountCount: len(bucket.accounts),
			Accounts:     make([]AccountDuplicateAccount, 0, len(bucket.accounts)),
		}
		sort.Slice(bucket.accounts, func(i, j int) bool { return bucket.accounts[i].ID < bucket.accounts[j].ID })
		for _, account := range bucket.accounts {
			group.Accounts = append(group.Accounts, duplicateAccountSummary(account))
		}
		groups = append(groups, group)
	}
	return groups
}

func duplicateAccountSummary(account duplicateCheckAccount) AccountDuplicateAccount {
	return AccountDuplicateAccount{
		ID:       account.ID,
		Name:     account.Name,
		Platform: account.Platform,
		Type:     account.Type,
		Status:   account.Status,
		ProxyID:  account.ProxyID,
		Groups:   append([]AccountDuplicateAccountGroup(nil), account.Groups...),
	}
}

func collectDuplicateAccountIDs(groups []AccountDuplicateGroup) map[int64]struct{} {
	ids := make(map[int64]struct{})
	for _, group := range groups {
		for _, account := range group.Accounts {
			ids[account.ID] = struct{}{}
		}
	}
	return ids
}

func sortDuplicateGroups(groups []AccountDuplicateGroup) {
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Severity != groups[j].Severity {
			return groups[i].Severity == AccountDuplicateSeverityStrong
		}
		if groups[i].KeyType != groups[j].KeyType {
			return groups[i].KeyType < groups[j].KeyType
		}
		if groups[i].Platform != groups[j].Platform {
			return groups[i].Platform < groups[j].Platform
		}
		return groups[i].MaskedValue < groups[j].MaskedValue
	})
}

func accountFieldString(fields map[string]any, key string) string {
	if fields == nil {
		return ""
	}
	value, ok := fields[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func normalizeStrongDuplicateValue(keyType string, value string) string {
	trimmed := strings.TrimSpace(value)
	if keyType == "api_key" || keyType == "refresh_token" {
		return trimmed
	}
	return strings.ToLower(trimmed)
}

func normalizeWeakDuplicateValue(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func maskDuplicateValue(value string) string {
	trimmed := strings.TrimSpace(value)
	hash := shortSHA256(trimmed)
	runes := []rune(trimmed)
	if len(runes) <= 4 {
		return "hash:" + hash
	}
	tailLen := 4
	if len(runes) > 12 {
		tailLen = 6
	}
	return "***" + string(runes[len(runes)-tailLen:]) + "#" + hash
}

func shortSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}

func copyAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
