package service

import (
	"crypto/rand"
	"fmt"
	"io"
	"strings"
)

type CodeCharacterSet string

const (
	CodeCharacterSetUppercase    CodeCharacterSet = "uppercase"
	CodeCharacterSetNumeric      CodeCharacterSet = "numeric"
	CodeCharacterSetAlphanumeric CodeCharacterSet = "alphanumeric"
	CodeCharacterSetHex          CodeCharacterSet = "hex"
)

const (
	codeAlphabetUppercase    = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	codeAlphabetNumeric      = "0123456789"
	codeAlphabetAlphanumeric = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeAlphabetHex          = "0123456789ABCDEF"
)

type CodeFormatConfig struct {
	Prefix       string           `json:"prefix"`
	CharacterSet CodeCharacterSet `json:"character_set"`
	Separator    string           `json:"separator"`
	GroupLength  int              `json:"group_length"`
	GroupCount   int              `json:"group_count"`
}

func DefaultRedeemCodeFormat() CodeFormatConfig {
	return CodeFormatConfig{
		CharacterSet: CodeCharacterSetHex,
		Separator:    "-",
		GroupLength:  8,
		GroupCount:   4,
	}
}

func DefaultCompactRedeemCodeFormat() CodeFormatConfig {
	return CodeFormatConfig{
		CharacterSet: CodeCharacterSetHex,
		GroupLength:  32,
		GroupCount:   1,
	}
}

func DefaultRedPacketCodeFormat() CodeFormatConfig {
	return CodeFormatConfig{
		CharacterSet: CodeCharacterSetHex,
		GroupLength:  24,
		GroupCount:   1,
	}
}

func (c CodeFormatConfig) Validate() error {
	if c.GroupLength < 1 || c.GroupLength > 32 {
		return fmt.Errorf("group_length must be between 1 and 32")
	}
	if c.GroupCount < 1 || c.GroupCount > 16 {
		return fmt.Errorf("group_count must be between 1 and 16")
	}
	if len(c.Prefix) > 16 {
		return fmt.Errorf("prefix must not exceed 16 ASCII characters")
	}
	if !isPrintableASCII(c.Prefix) {
		return fmt.Errorf("prefix must contain printable ASCII characters only")
	}
	if len(c.Separator) > 1 || !isPrintableASCII(c.Separator) {
		return fmt.Errorf("separator must be empty or one printable ASCII character")
	}
	if c.Separator != "" && strings.Contains(c.Prefix, c.Separator) {
		return fmt.Errorf("prefix must not contain the separator")
	}
	if _, err := c.alphabet(); err != nil {
		return err
	}
	if c.outputLength() > 128 {
		return fmt.Errorf("formatted code must not exceed 128 characters")
	}
	return nil
}

func (c CodeFormatConfig) Generate() (string, error) {
	return c.generate(rand.Reader)
}

func (c CodeFormatConfig) generate(reader io.Reader) (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}
	alphabet, _ := c.alphabet()
	groups := make([]string, c.GroupCount)
	for i := range groups {
		group, err := secureRandomCharacters(reader, alphabet, c.GroupLength)
		if err != nil {
			return "", fmt.Errorf("generate secure code: %w", err)
		}
		groups[i] = group
	}
	body := strings.Join(groups, c.Separator)
	if c.Prefix == "" {
		return body, nil
	}
	if c.Separator == "" {
		return c.Prefix + body, nil
	}
	return c.Prefix + c.Separator + body, nil
}

func (c CodeFormatConfig) alphabet() (string, error) {
	switch c.CharacterSet {
	case CodeCharacterSetUppercase:
		return codeAlphabetUppercase, nil
	case CodeCharacterSetNumeric:
		return codeAlphabetNumeric, nil
	case CodeCharacterSetAlphanumeric:
		return codeAlphabetAlphanumeric, nil
	case CodeCharacterSetHex:
		return codeAlphabetHex, nil
	default:
		return "", fmt.Errorf("unsupported character_set %q", c.CharacterSet)
	}
}

func (c CodeFormatConfig) outputLength() int {
	length := c.GroupLength*c.GroupCount + len(c.Separator)*(c.GroupCount-1)
	if c.Prefix != "" {
		length += len(c.Prefix)
		if c.Separator != "" {
			length += len(c.Separator)
		}
	}
	return length
}

func secureRandomCharacters(reader io.Reader, alphabet string, length int) (string, error) {
	if reader == nil {
		return "", fmt.Errorf("random reader is required")
	}
	if len(alphabet) < 2 || len(alphabet) > 256 {
		return "", fmt.Errorf("alphabet size must be between 2 and 256")
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
