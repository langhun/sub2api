package settings

import (
	"context"
	"encoding/json"
	"fmt"
)

// PublicSettingsSource supplies the upstream-owned public settings payload
// before Overlay-owned values are projected into the same JSON contract.
type PublicSettingsSource interface {
	GetPublicSettingsForInjection(context.Context) (any, error)
}

// InjectionProvider is the fixed bridge used by the embedded SPA. It keeps
// public Overlay values sourced from Registry rather than shared settings.
type InjectionProvider struct {
	source   PublicSettingsSource
	registry *Registry
}

func NewInjectionProvider(source PublicSettingsSource, registry *Registry) *InjectionProvider {
	return &InjectionProvider{source: source, registry: registry}
}

func (p *InjectionProvider) GetPublicSettingsForInjection(ctx context.Context) (any, error) {
	if p == nil || p.source == nil || p.registry == nil {
		return nil, fmt.Errorf("public settings injection provider is not configured")
	}
	base, err := p.source.GetPublicSettingsForInjection(ctx)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(base)
	if err != nil {
		return nil, fmt.Errorf("marshal upstream public settings: %w", err)
	}
	merged := make(map[string]any)
	if err := json.Unmarshal(encoded, &merged); err != nil {
		return nil, fmt.Errorf("decode upstream public settings: %w", err)
	}
	overlay, err := p.registry.Public(ctx)
	if err != nil {
		return nil, err
	}
	for key, value := range overlay {
		merged[key] = value
	}
	return merged, nil
}
