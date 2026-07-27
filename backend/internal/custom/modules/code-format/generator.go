// Package codeformat provides the runtime code-generation capability owned by
// the Overlay code-format module.
package codeformat

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/code-format/settings"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
)

type Generator struct{ registry *customsettings.Registry }

func NewGenerator(registry *customsettings.Registry) *Generator {
	return &Generator{registry: registry}
}

func (g *Generator) GenerateCode(ctx context.Context, codeType string) (string, error) {
	if g == nil || g.registry == nil {
		return "", fmt.Errorf("code format settings registry is required")
	}
	snapshot, err := g.registry.Read(ctx)
	if err != nil {
		return "", fmt.Errorf("read code format settings: %w", err)
	}
	return snapshot.CodeFormat.RedeemFormat(codeType).Generate()
}

// GenerateDefaultRedeemCode preserves the legacy random redeem-code shape.
func (g *Generator) GenerateDefaultRedeemCode(context.Context) (string, error) {
	return settings.DefaultRedeemFormat().Generate()
}

func (g *Generator) GenerateRedPacketCode(ctx context.Context) (string, error) {
	if g == nil || g.registry == nil {
		return "", fmt.Errorf("code format settings registry is required")
	}
	snapshot, err := g.registry.Read(ctx)
	if err != nil {
		return "", fmt.Errorf("read code format settings: %w", err)
	}
	return snapshot.CodeFormat.RedPacket.Generate()
}
