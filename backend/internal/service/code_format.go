package service

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	codeFormatMaxTotalLength = 32
	codeFormatMaxPartLength  = 16
)

type CodeFormatSettings struct {
	Prefix       string `json:"prefix"`
	Suffix       string `json:"suffix"`
	RandomLength int    `json:"random_length"`
	Separator    string `json:"separator"`
	GroupSize    int    `json:"group_size"`
}

func DefaultRegistrationInvitationCodeFormat() CodeFormatSettings {
	return CodeFormatSettings{
		Prefix:       "DG",
		Suffix:       "",
		RandomLength: 6,
		Separator:    "-",
		GroupSize:    0,
	}
}

func DefaultRedeemCodeFormat() CodeFormatSettings {
	return CodeFormatSettings{
		Prefix:       "",
		Suffix:       "",
		RandomLength: 16,
		Separator:    "-",
		GroupSize:    4,
	}
}

func DefaultAffiliateCodeFormat() CodeFormatSettings {
	return CodeFormatSettings{
		Prefix:       "",
		Suffix:       "",
		RandomLength: 12,
		Separator:    "",
		GroupSize:    0,
	}
}

func normalizeCodeFormatSettings(cfg CodeFormatSettings) CodeFormatSettings {
	cfg.Prefix = strings.ToUpper(strings.TrimSpace(cfg.Prefix))
	cfg.Suffix = strings.ToUpper(strings.TrimSpace(cfg.Suffix))
	cfg.Separator = strings.TrimSpace(cfg.Separator)
	return cfg
}

func isCodeFormatTokenValid(raw string) bool {
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '-' && ch != '_' {
			return false
		}
	}
	return true
}

func ValidateCodeFormatSettings(cfg CodeFormatSettings) error {
	cfg = normalizeCodeFormatSettings(cfg)
	if cfg.RandomLength <= 0 {
		return fmt.Errorf("random length must be greater than 0")
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
	if cfg.GroupSize < 0 {
		return fmt.Errorf("group size must be greater than or equal to 0")
	}
	if cfg.GroupSize > 0 && cfg.GroupSize > cfg.RandomLength {
		return fmt.Errorf("group size cannot exceed random length")
	}
	if len(renderCodeWithFormat(strings.Repeat("A", cfg.RandomLength), cfg)) > codeFormatMaxTotalLength {
		return fmt.Errorf("formatted code is too long")
	}
	return nil
}

func ParseCodeFormatSettings(raw string, defaults CodeFormatSettings) CodeFormatSettings {
	cfg := defaults
	if strings.TrimSpace(raw) == "" {
		return cfg
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return defaults
	}
	cfg = normalizeCodeFormatSettings(cfg)
	if err := ValidateCodeFormatSettings(cfg); err != nil {
		return defaults
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
	randomPart := core
	if cfg.Separator != "" && cfg.GroupSize > 0 && cfg.GroupSize < len(core) {
		parts := make([]string, 0, (len(core)+cfg.GroupSize-1)/cfg.GroupSize)
		for start := 0; start < len(core); start += cfg.GroupSize {
			end := start + cfg.GroupSize
			if end > len(core) {
				end = len(core)
			}
			parts = append(parts, core[start:end])
		}
		randomPart = strings.Join(parts, cfg.Separator)
	}

	parts := make([]string, 0, 3)
	if cfg.Prefix != "" {
		parts = append(parts, cfg.Prefix)
	}
	parts = append(parts, randomPart)
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
	if len(charset) == 0 {
		return "", fmt.Errorf("charset is empty")
	}

	buf := make([]byte, cfg.RandomLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = charset[int(buf[i])%len(charset)]
	}
	return renderCodeWithFormat(string(buf), cfg), nil
}

func IsCodeMatchingFormat(code string, cfg CodeFormatSettings) bool {
	cfg = normalizeCodeFormatSettings(cfg)
	if err := ValidateCodeFormatSettings(cfg); err != nil {
		return false
	}

	code = strings.ToUpper(strings.TrimSpace(code))
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
		minParts := 1
		if cfg.Prefix != "" {
			minParts++
		}
		if cfg.Suffix != "" {
			minParts++
		}
		if len(parts) < minParts {
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
		remainder = strings.Join(parts, "")
	}

	if len(remainder) != cfg.RandomLength {
		return false
	}
	for i := 0; i < len(remainder); i++ {
		ch := remainder[i]
		if (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') {
			return false
		}
	}

	if cfg.Separator != "" && cfg.GroupSize > 0 {
		expected := renderCodeWithFormat(remainder, cfg)
		return expected == code
	}
	return true
}
