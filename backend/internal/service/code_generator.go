package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
)

// CodeGenerator is the only formatting capability required by core services.
// Its implementation is supplied by the Overlay composition root.
type CodeGenerator interface {
	GenerateCode(context.Context, string) (string, error)
	GenerateDefaultRedeemCode(context.Context) (string, error)
}

// defaultCodeGenerator preserves the legacy behavior for direct service
// construction outside the application composition root. It deliberately owns
// only the two historical fallback shapes; configured formats stay in custom.
type defaultCodeGenerator struct{}

func (defaultCodeGenerator) GenerateCode(context.Context, string) (string, error) {
	return generateDefaultCode(32, 1, "")
}

func (defaultCodeGenerator) GenerateDefaultRedeemCode(context.Context) (string, error) {
	return generateDefaultCode(8, 4, "-")
}

const defaultCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func generateDefaultCode(groupLength, groupCount int, separator string) (string, error) {
	groups := make([]string, groupCount)
	for i := range groups {
		group, err := generateDefaultCodeCharacters(rand.Reader, groupLength)
		if err != nil {
			return "", fmt.Errorf("generate secure code: %w", err)
		}
		groups[i] = group
	}
	return strings.Join(groups, separator), nil
}

func generateDefaultCodeCharacters(reader io.Reader, length int) (string, error) {
	limit := 256 - (256 % len(defaultCodeAlphabet))
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
			result = append(result, defaultCodeAlphabet[int(value)%len(defaultCodeAlphabet)])
			if len(result) == length {
				break
			}
		}
	}
	return string(result), nil
}
