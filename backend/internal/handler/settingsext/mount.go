// Package settingsext defines the core settings extension seam. Implementations
// own their fields and persistence rules; core handlers only mount JSON payloads.
package settingsext

import (
	"context"
	"encoding/json"
	"fmt"
)

type Mount interface {
	Admin(context.Context) (map[string]any, error)
	Public(context.Context) (map[string]any, error)
	ValidateUpdate(context.Context, json.RawMessage) (bool, error)
	ApplyUpdate(context.Context, json.RawMessage) error
}

type EmptyMount struct{}

func (EmptyMount) Admin(context.Context) (map[string]any, error)  { return nil, nil }
func (EmptyMount) Public(context.Context) (map[string]any, error) { return nil, nil }
func (EmptyMount) ValidateUpdate(context.Context, json.RawMessage) (bool, error) {
	return false, nil
}
func (EmptyMount) ApplyUpdate(context.Context, json.RawMessage) error { return nil }

func Merge(payload any, extension map[string]any) (map[string]any, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode settings payload: %w", err)
	}
	merged := make(map[string]any)
	if err := json.Unmarshal(encoded, &merged); err != nil {
		return nil, fmt.Errorf("decode settings payload: %w", err)
	}
	for key, value := range extension {
		// A mounted extension owns its declared keys. Overriding lets an overlay
		// migrate a formerly core-owned field without changing the JSON contract.
		merged[key] = value
	}
	return merged, nil
}
