package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/handler/settingsext"
)

var _ settingsext.Mount = (*HandlerMount)(nil)

// HandlerMount owns the custom settings JSON contract behind the core handler
// extension seam. Core code does not import this package or its feature keys.
type HandlerMount struct {
	registry *Registry
}

func NewHandlerMount(registry *Registry) *HandlerMount {
	return &HandlerMount{registry: registry}
}

func (m *HandlerMount) Admin(ctx context.Context) (map[string]any, error) {
	if m == nil || m.registry == nil {
		return nil, fmt.Errorf("custom settings registry is not configured")
	}
	settings, err := m.registry.Compatibility(ctx)
	if err != nil {
		return nil, err
	}
	return mapFrom(settings)
}

func (m *HandlerMount) Public(ctx context.Context) (map[string]any, error) {
	if m == nil || m.registry == nil {
		return nil, fmt.Errorf("custom settings registry is not configured")
	}
	settings, err := m.registry.Compatibility(ctx)
	if err != nil {
		return nil, err
	}
	return mapFrom(settings.Public())
}

func (m *HandlerMount) ValidateUpdate(ctx context.Context, payload json.RawMessage) (bool, error) {
	patch, err := patchFrom(payload)
	if err != nil {
		return false, err
	}
	if !patch.HasChanges() {
		return false, nil
	}
	previous, err := m.snapshot(ctx)
	if err != nil {
		return false, err
	}
	return true, m.registry.Validate(patch.Apply(previous))
}

func (m *HandlerMount) ApplyUpdate(ctx context.Context, payload json.RawMessage) error {
	patch, err := patchFrom(payload)
	if err != nil {
		return err
	}
	if !patch.HasChanges() {
		return nil
	}
	previous, err := m.snapshot(ctx)
	if err != nil {
		return err
	}
	return m.registry.Write(ctx, patch.Apply(previous))
}

func (m *HandlerMount) snapshot(ctx context.Context) (Snapshot, error) {
	if m == nil || m.registry == nil {
		return Snapshot{}, fmt.Errorf("custom settings registry is not configured")
	}
	return m.registry.Read(ctx)
}

func patchFrom(payload json.RawMessage) (Patch, error) {
	var patch Patch
	if err := json.Unmarshal(payload, &patch); err != nil {
		return Patch{}, fmt.Errorf("decode custom settings patch: %w", err)
	}
	return patch, nil
}

func mapFrom(value any) (map[string]any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any)
	if err := json.Unmarshal(encoded, &result); err != nil {
		return nil, err
	}
	return result, nil
}
