package bepusdt

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

func signMap(values map[string]any, token string) string {
	keys := make([]string, 0, len(values))
	for k, v := range values {
		if k != "signature" && v != nil && fmt.Sprint(v) != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+fmt.Sprint(values[k]))
	}
	h := md5.Sum([]byte(strings.Join(parts, "&") + token))
	return hex.EncodeToString(h[:])
}
