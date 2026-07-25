// Package settings owns the persisted configuration for the Overlay brand home.
package settings

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
)

const KeyDefaultHomepage = "default_homepage"

const (
	HomepageDefault = "default"
	HomepageDino    = "dino"
)

type Config struct {
	DefaultHomepage string
}

func Default() Config {
	return Config{DefaultHomepage: HomepageDefault}
}

type Reader struct{ store contract.Store }

func New(store contract.Store) *Reader { return &Reader{store: store} }

func (r *Reader) Read(ctx context.Context) (Config, error) {
	if r == nil || r.store == nil {
		return Config{}, fmt.Errorf("brand home settings store is required")
	}
	values, err := r.store.GetMultiple(ctx, Keys())
	if err != nil {
		return Config{}, fmt.Errorf("read brand home settings: %w", err)
	}
	return FromValues(values), nil
}

func (r *Reader) Write(ctx context.Context, config Config) error {
	values, err := Values(config)
	if err != nil {
		return err
	}
	if r == nil || r.store == nil {
		return fmt.Errorf("brand home settings store is required")
	}
	return r.store.SetMultiple(ctx, values)
}

func Keys() []string { return []string{KeyDefaultHomepage} }

func FromValues(values map[string]string) Config {
	return Config{DefaultHomepage: normalize(values[KeyDefaultHomepage])}
}

func Values(config Config) (map[string]string, error) {
	if err := Validate(config); err != nil {
		return nil, err
	}
	return map[string]string{KeyDefaultHomepage: config.DefaultHomepage}, nil
}

func Validate(config Config) error {
	if config.DefaultHomepage != HomepageDefault && config.DefaultHomepage != HomepageDino {
		return fmt.Errorf("default_homepage must be %q or %q", HomepageDefault, HomepageDino)
	}
	return nil
}

func Public(config Config) map[string]any {
	return map[string]any{KeyDefaultHomepage: config.DefaultHomepage}
}

func normalize(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), HomepageDino) {
		return HomepageDino
	}
	return HomepageDefault
}
