package service

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	codeFormatMaxTotalLength   = 32
	codeFormatMaxPartLength    = 16
	codeFormatMaxGroupCount    = 16
	codeFormatMaxCharsPerGroup = 16

	CodeFormatCharsetDigits  = "digits"
	CodeFormatCharsetLetters = "letters"
	CodeFormatCharsetMixed   = "mixed"

	CodeFormatLetterCaseUpper = "upper"
	CodeFormatLetterCaseLower = "lower"
)

type CodeFormatSettings struct {
	Prefix        string `json:"prefix"`
	Suffix        string `json:"suffix"`
	RandomLength  int    `json:"random_length"`
	Separator     string `json:"separator"`
	GroupSize     int    `json:"group_size"`
	GroupCount    int    `json:"group_count,omitempty"`
	CharsPerGroup int    `json:"chars_per_group,omitempty"`
	Charset       string `json:"charset,omitempty"`
	LetterCase    string `json:"letter_case,omitempty"`
}

func DefaultRegistrationInvitationCodeFormat() CodeFormatSettings {
	return CodeFormatSettings{
		Prefix:        "DG",
		Suffix:        "",
		RandomLength:  6,
		Separator:     "-",
		GroupSize:     6,
		GroupCount:    1,
		CharsPerGroup: 6,
		Charset:       CodeFormatCharsetMixed,
		LetterCase:    CodeFormatLetterCaseUpper,
	}
}

func DefaultRedeemCodeFormat() CodeFormatSettings {
	return CodeFormatSettings{
		Prefix:        "",
		Suffix:        "",
		RandomLength:  16,
		Separator:     "-",
		GroupSize:     4,
		GroupCount:    4,
		CharsPerGroup: 4,
		Charset:       CodeFormatCharsetMixed,
		LetterCase:    CodeFormatLetterCaseUpper,
	}
}

func DefaultAffiliateCodeFormat() CodeFormatSettings {
	return CodeFormatSettings{
		Prefix:        "",
		Suffix:        "",
		RandomLength:  12,
		Separator:     "",
		GroupSize:     12,
		GroupCount:    1,
		CharsPerGroup: 12,
		Charset:       CodeFormatCharsetMixed,
		LetterCase:    CodeFormatLetterCaseUpper,
	}
}

func normalizeCodeFormatCharset(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case CodeFormatCharsetDigits:
		return CodeFormatCharsetDigits
	case CodeFormatCharsetLetters:
		return CodeFormatCharsetLetters
	default:
		return CodeFormatCharsetMixed
	}
}

func normalizeCodeFormatLetterCase(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case CodeFormatLetterCaseLower:
		return CodeFormatLetterCaseLower
	default:
		return CodeFormatLetterCaseUpper
	}
}

func applyCodeLetterCase(raw, letterCase string) string {
	switch normalizeCodeFormatLetterCase(letterCase) {
	case CodeFormatLetterCaseLower:
		return strings.ToLower(raw)
	default:
		return strings.ToUpper(raw)
	}
}

func normalizeCodeFormatSettings(cfg CodeFormatSettings) CodeFormatSettings {
	cfg.Charset = normalizeCodeFormatCharset(cfg.Charset)
	cfg.LetterCase = normalizeCodeFormatLetterCase(cfg.LetterCase)
	cfg.Prefix = applyCodeLetterCase(strings.TrimSpace(cfg.Prefix), cfg.LetterCase)
	cfg.Suffix = applyCodeLetterCase(strings.TrimSpace(cfg.Suffix), cfg.LetterCase)
	cfg.Separator = strings.TrimSpace(cfg.Separator)

	if cfg.GroupCount > 0 || cfg.CharsPerGroup > 0 {
		if cfg.GroupCount <= 0 {
			cfg.GroupCount = 1
		}
		if cfg.CharsPerGroup <= 0 {
			switch {
			case cfg.GroupSize > 0:
				cfg.CharsPerGroup = cfg.GroupSize
			case cfg.RandomLength > 0:
				cfg.CharsPerGroup = cfg.RandomLength
			default:
				cfg.CharsPerGroup = 1
			}
		}
		cfg.GroupSize = cfg.CharsPerGroup
		cfg.RandomLength = cfg.GroupCount * cfg.CharsPerGroup
		return cfg
	}

	if cfg.GroupSize > 0 && cfg.RandomLength > 0 {
		cfg.GroupCount = (cfg.RandomLength + cfg.GroupSize - 1) / cfg.GroupSize
		cfg.CharsPerGroup = cfg.GroupSize
		return cfg
	}

	if cfg.RandomLength > 0 {
		cfg.GroupCount = 1
		cfg.CharsPerGroup = cfg.RandomLength
		cfg.GroupSize = cfg.RandomLength
	}

	return cfg
}

func NormalizeCodeValueWithFormat(raw string, cfg CodeFormatSettings) string {
	cfg = normalizeCodeFormatSettings(cfg)
	return applyCodeLetterCase(strings.TrimSpace(raw), cfg.LetterCase)
}

func isCodeFormatTokenValid(raw string) bool {
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if (ch < 'A' || ch > 'Z') &&
			(ch < 'a' || ch > 'z') &&
			(ch < '0' || ch > '9') &&
			ch != '-' &&
			ch != '_' {
			return false
		}
	}
	return true
}

func buildCodeCharset(cfg CodeFormatSettings, baseCharset []byte) ([]byte, error) {
	filtered := make([]byte, 0, len(baseCharset))
	seen := make(map[byte]struct{}, len(baseCharset))
	for _, ch := range baseCharset {
		switch cfg.Charset {
		case CodeFormatCharsetDigits:
			if ch < '0' || ch > '9' {
				continue
			}
		case CodeFormatCharsetLetters:
			if (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') {
				continue
			}
		default:
			if (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') {
				continue
			}
		}

		if ch >= 'a' && ch <= 'z' {
			if cfg.LetterCase == CodeFormatLetterCaseUpper {
				ch = ch - 'a' + 'A'
			}
		} else if ch >= 'A' && ch <= 'Z' {
			if cfg.LetterCase == CodeFormatLetterCaseLower {
				ch = ch - 'A' + 'a'
			}
		}

		if _, ok := seen[ch]; ok {
			continue
		}
		seen[ch] = struct{}{}
		filtered = append(filtered, ch)
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("charset does not contain any usable characters")
	}
	return filtered, nil
}

func sampleCodeCore(cfg CodeFormatSettings) string {
	var unit string
	switch cfg.Charset {
	case CodeFormatCharsetDigits:
		unit = "1"
	default:
		if cfg.LetterCase == CodeFormatLetterCaseLower {
			unit = "a"
		} else {
			unit = "A"
		}
	}
	return strings.Repeat(unit, cfg.RandomLength)
}

func ValidateCodeFormatSettings(cfg CodeFormatSettings) error {
	cfg = normalizeCodeFormatSettings(cfg)
	if cfg.GroupCount <= 0 {
		return fmt.Errorf("group count must be greater than 0")
	}
	if cfg.GroupCount > codeFormatMaxGroupCount {
		return fmt.Errorf("group count is too large")
	}
	if cfg.CharsPerGroup <= 0 {
		return fmt.Errorf("chars per group must be greater than 0")
	}
	if cfg.CharsPerGroup > codeFormatMaxCharsPerGroup {
		return fmt.Errorf("chars per group is too large")
	}
	if cfg.RandomLength <= 0 {
		return fmt.Errorf("random length must be greater than 0")
	}
	if cfg.GroupCount*cfg.CharsPerGroup != cfg.RandomLength {
		return fmt.Errorf("random length must equal group count * chars per group")
	}
	if len(cfg.Prefix) > codeFormatMaxPartLength {
		return fmt.Errorf("prefix is too long")
	}
	if len(cfg.Suffix) > codeFormatMaxPartLength {
		return fmt.Errorf("suffix is too long")
	}
	if !isCodeFormatTokenValid(cfg.Prefix) {
		return fmt.Errorf("prefix contains unsupported characters")
	}
	if !isCodeFormatTokenValid(cfg.Suffix) {
		return fmt.Errorf("suffix contains unsupported characters")
	}
	if cfg.Separator != "" && cfg.Separator != "-" && cfg.Separator != "_" {
		return fmt.Errorf("separator must be empty, '-' or '_'")
	}
	if cfg.Separator != "" &&
		(strings.Contains(cfg.Prefix, cfg.Separator) || strings.Contains(cfg.Suffix, cfg.Separator)) {
		return fmt.Errorf("prefix and suffix cannot contain the separator")
	}
	if cfg.Charset != CodeFormatCharsetDigits &&
		cfg.Charset != CodeFormatCharsetLetters &&
		cfg.Charset != CodeFormatCharsetMixed {
		return fmt.Errorf("charset must be digits, letters or mixed")
	}
	if cfg.LetterCase != CodeFormatLetterCaseUpper &&
		cfg.LetterCase != CodeFormatLetterCaseLower {
		return fmt.Errorf("letter case must be upper or lower")
	}
	if len(renderCodeWithFormat(sampleCodeCore(cfg), cfg)) > codeFormatMaxTotalLength {
		return fmt.Errorf("formatted code is too long")
	}
	return nil
}

func ParseCodeFormatSettings(raw string, defaults CodeFormatSettings) CodeFormatSettings {
	cfg := defaults
	if strings.TrimSpace(raw) == "" {
		return normalizeCodeFormatSettings(cfg)
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return normalizeCodeFormatSettings(defaults)
	}
	cfg = normalizeCodeFormatSettings(cfg)
	if err := ValidateCodeFormatSettings(cfg); err != nil {
		return normalizeCodeFormatSettings(defaults)
	}
	return cfg
}

func MarshalCodeFormatSettings(cfg CodeFormatSettings) (string, error) {
	cfg = normalizeCodeFormatSettings(cfg)
	if err := ValidateCodeFormatSettings(cfg); err != nil {
		return "", err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func mustMarshalCodeFormatDefaults(cfg CodeFormatSettings) string {
	value, err := MarshalCodeFormatSettings(cfg)
	if err != nil {
		panic(err)
	}
	return value
}

func renderCodeWithFormat(core string, cfg CodeFormatSettings) string {
	cfg = normalizeCodeFormatSettings(cfg)
	randomParts := []string{core}
	if cfg.GroupCount > 1 && cfg.CharsPerGroup > 0 {
		randomParts = make([]string, 0, cfg.GroupCount)
		for start := 0; start < len(core); start += cfg.CharsPerGroup {
			end := start + cfg.CharsPerGroup
			if end > len(core) {
				end = len(core)
			}
			randomParts = append(randomParts, core[start:end])
		}
	}

	parts := make([]string, 0, len(randomParts)+2)
	if cfg.Prefix != "" {
		parts = append(parts, cfg.Prefix)
	}
	parts = append(parts, randomParts...)
	if cfg.Suffix != "" {
		parts = append(parts, cfg.Suffix)
	}

	if cfg.Separator == "" {
		return strings.Join(parts, "")
	}
	return strings.Join(parts, cfg.Separator)
}

func GenerateCodeWithFormat(cfg CodeFormatSettings, charset []byte) (string, error) {
	cfg = normalizeCodeFormatSettings(cfg)
	if err := ValidateCodeFormatSettings(cfg); err != nil {
		return "", err
	}
	filteredCharset, err := buildCodeCharset(cfg, charset)
	if err != nil {
		return "", err
	}

	buf := make([]byte, cfg.RandomLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = filteredCharset[int(buf[i])%len(filteredCharset)]
	}
	return renderCodeWithFormat(string(buf), cfg), nil
}

func IsCodeMatchingFormat(code string, cfg CodeFormatSettings) bool {
	cfg = normalizeCodeFormatSettings(cfg)
	if err := ValidateCodeFormatSettings(cfg); err != nil {
		return false
	}

	code = NormalizeCodeValueWithFormat(code, cfg)
	remainder := code

	if cfg.Separator == "" {
		if cfg.Prefix != "" {
			if !strings.HasPrefix(remainder, cfg.Prefix) {
				return false
			}
			remainder = remainder[len(cfg.Prefix):]
		}
		if cfg.Suffix != "" {
			if !strings.HasSuffix(remainder, cfg.Suffix) {
				return false
			}
			remainder = remainder[:len(remainder)-len(cfg.Suffix)]
		}
	} else {
		parts := strings.Split(remainder, cfg.Separator)
		expectedParts := cfg.GroupCount
		if cfg.Prefix != "" {
			expectedParts++
		}
		if cfg.Suffix != "" {
			expectedParts++
		}
		if len(parts) != expectedParts {
			return false
		}
		if cfg.Prefix != "" {
			if parts[0] != cfg.Prefix {
				return false
			}
			parts = parts[1:]
		}
		if cfg.Suffix != "" {
			last := len(parts) - 1
			if last < 0 || parts[last] != cfg.Suffix {
				return false
			}
			parts = parts[:last]
		}
		for _, part := range parts {
			if len(part) != cfg.CharsPerGroup {
				return false
			}
		}
		remainder = strings.Join(parts, "")
	}

	if len(remainder) != cfg.RandomLength {
		return false
	}

	for i := 0; i < len(remainder); i++ {
		ch := remainder[i]
		switch cfg.Charset {
		case CodeFormatCharsetDigits:
			if ch < '0' || ch > '9' {
				return false
			}
		case CodeFormatCharsetLetters:
			if cfg.LetterCase == CodeFormatLetterCaseLower {
				if ch < 'a' || ch > 'z' {
					return false
				}
			} else if ch < 'A' || ch > 'Z' {
				return false
			}
		default:
			if cfg.LetterCase == CodeFormatLetterCaseLower {
				if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') {
					return false
				}
			} else if (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') {
				return false
			}
		}
	}

	if cfg.Separator != "" {
		return renderCodeWithFormat(remainder, cfg) == code
	}
	return true
}
