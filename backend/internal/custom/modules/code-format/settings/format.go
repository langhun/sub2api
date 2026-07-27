package settings

import (
	"crypto/rand"
	"fmt"
	"io"
	"strings"
)

type CharacterSet string

const (
	CharacterSetUppercase    CharacterSet = "uppercase"
	CharacterSetNumeric      CharacterSet = "numeric"
	CharacterSetAlphanumeric CharacterSet = "alphanumeric"
	characterSetLegacyHex    CharacterSet = "hex"
)

const (
	codeAlphabetUppercase    = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	codeAlphabetNumeric      = "0123456789"
	codeAlphabetAlphanumeric = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

type Format struct {
	Prefix       string       `json:"prefix"`
	CharacterSet CharacterSet `json:"character_set"`
	Separator    string       `json:"separator"`
	GroupLength  int          `json:"group_length"`
	GroupCount   int          `json:"group_count"`
}

func DefaultCompactFormat() Format {
	return Format{CharacterSet: CharacterSetAlphanumeric, GroupLength: 32, GroupCount: 1}
}

// DefaultRedeemFormat preserves the long-standing random redeem-code shape.
func DefaultRedeemFormat() Format {
	return Format{CharacterSet: CharacterSetAlphanumeric, Separator: "-", GroupLength: 8, GroupCount: 4}
}

func DefaultRedPacketFormat() Format {
	return Format{CharacterSet: CharacterSetAlphanumeric, GroupLength: 24, GroupCount: 1}
}

func (f Format) Validate() error {
	if f.GroupLength < 1 || f.GroupLength > 32 {
		return fmt.Errorf("group_length must be between 1 and 32")
	}
	if f.GroupCount < 1 || f.GroupCount > 16 {
		return fmt.Errorf("group_count must be between 1 and 16")
	}
	if len(f.Prefix) > 16 {
		return fmt.Errorf("prefix must not exceed 16 ASCII characters")
	}
	if !isPrintableASCII(f.Prefix) {
		return fmt.Errorf("prefix must contain printable ASCII characters only")
	}
	if len(f.Separator) > 1 || !isPrintableASCII(f.Separator) {
		return fmt.Errorf("separator must be empty or one printable ASCII character")
	}
	if f.Separator != "" && strings.Contains(f.Prefix, f.Separator) {
		return fmt.Errorf("prefix must not contain the separator")
	}
	if _, err := f.alphabet(); err != nil {
		return err
	}
	if f.outputLength() > 128 {
		return fmt.Errorf("formatted code must not exceed 128 characters")
	}
	return nil
}

func (f Format) Generate() (string, error) { return f.generate(rand.Reader) }

func (f Format) generate(reader io.Reader) (string, error) {
	if err := f.Validate(); err != nil {
		return "", err
	}
	alphabet, _ := f.alphabet()
	groups := make([]string, f.GroupCount)
	for i := range groups {
		group, err := secureRandomCharacters(reader, alphabet, f.GroupLength)
		if err != nil {
			return "", fmt.Errorf("generate secure code: %w", err)
		}
		groups[i] = group
	}
	body := strings.Join(groups, f.Separator)
	if f.Prefix == "" {
		return body, nil
	}
	if f.Separator == "" {
		return f.Prefix + body, nil
	}
	return f.Prefix + f.Separator + body, nil
}

func (f Format) alphabet() (string, error) {
	switch f.CharacterSet {
	case CharacterSetUppercase:
		return codeAlphabetUppercase, nil
	case CharacterSetNumeric:
		return codeAlphabetNumeric, nil
	case CharacterSetAlphanumeric:
		return codeAlphabetAlphanumeric, nil
	default:
		return "", fmt.Errorf("unsupported character_set %q", f.CharacterSet)
	}
}

func (f Format) outputLength() int {
	length := f.GroupLength*f.GroupCount + len(f.Separator)*(f.GroupCount-1)
	if f.Prefix != "" {
		length += len(f.Prefix)
		if f.Separator != "" {
			length += len(f.Separator)
		}
	}
	return length
}

func secureRandomCharacters(reader io.Reader, alphabet string, length int) (string, error) {
	if reader == nil {
		return "", fmt.Errorf("random reader is required")
	}
	limit := 256 - (256 % len(alphabet))
	result := make([]byte, 0, length)
	buffer := make([]byte, length*2+1)
	for len(result) < length {
		if _, err := io.ReadFull(reader, buffer); err != nil {
			return "", err
		}
		for _, value := range buffer {
			if int(value) >= limit {
				continue
			}
			result = append(result, alphabet[int(value)%len(alphabet)])
			if len(result) == length {
				break
			}
		}
	}
	return string(result), nil
}

func isPrintableASCII(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x21 || value[i] > 0x7e {
			return false
		}
	}
	return true
}
