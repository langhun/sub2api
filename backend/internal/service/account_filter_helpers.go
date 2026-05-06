package service

import (
	"sort"
	"strings"
)

func dedupePositiveInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeStringFilters(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}

func normalizeAccountStatusFilters(values []string) []string {
	normalized := normalizeStringFilters(values)
	seen := make(map[string]struct{}, len(normalized)+2)
	out := make([]string, 0, len(normalized)+2)
	for _, status := range normalized {
		if _, ok := seen[status]; ok {
			continue
		}
		seen[status] = struct{}{}
		out = append(out, status)
		if status == "inactive" {
			if _, ok := seen[StatusDisabled]; !ok {
				seen[StatusDisabled] = struct{}{}
				out = append(out, StatusDisabled)
			}
		}
	}
	sort.Strings(out)
	return out
}
