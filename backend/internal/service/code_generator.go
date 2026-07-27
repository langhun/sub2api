package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// CodeGenerator is the narrow formatting port that an Overlay may configure at
// application startup. Core services remain functional without one.
type CodeGenerator interface {
	GenerateCode(context.Context, string) (string, error)
	GenerateDefaultRedeemCode(context.Context) (string, error)
}

// defaultCodeGenerator preserves the upstream behavior when no Overlay is
// configured. Configured formats are owned by custom.
type defaultCodeGenerator struct{}

func (defaultCodeGenerator) GenerateCode(context.Context, string) (string, error) {
	return generateCompactRedeemCode()
}

func (defaultCodeGenerator) GenerateDefaultRedeemCode(context.Context) (string, error) {
	code, err := generateCompactRedeemCode()
	if err != nil {
		return "", err
	}
	code = strings.ToUpper(code)
	return strings.Join([]string{
		code[0:8],
		code[8:16],
		code[16:24],
		code[24:32],
	}, "-"), nil
}

func generateCompactRedeemCode() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ConfigureCodeGenerator mounts an optional implementation during application
// composition. It intentionally depends only on the core port, never on a
// custom package.
func ConfigureCodeGenerator(redeem *RedeemService, admin AdminService, generator CodeGenerator) {
	if generator == nil {
		return
	}
	if redeem != nil {
		redeem.SetCodeGenerator(generator)
	}
	if configurable, ok := admin.(interface{ setCodeGenerator(CodeGenerator) }); ok {
		configurable.setCodeGenerator(generator)
	}
}
