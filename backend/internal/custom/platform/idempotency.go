// Package platform contains narrowly-scoped adapters from Overlay modules to
// generic core capabilities. It is the only custom-layer owner of the global
// idempotency coordinator.
package platform

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// IdempotencyOptions is the module-neutral representation of an idempotent
// HTTP operation. It intentionally mirrors the established coordinator fields
// so scope, key, and replay semantics remain unchanged.
type IdempotencyOptions struct {
	Scope          string
	ActorScope     string
	Method         string
	Route          string
	IdempotencyKey string
	Payload        any
	TTL            time.Duration
	RequireKey     bool
}

type IdempotencyResult struct {
	Data     any
	Replayed bool
}

// IdempotencyCoordinator is the platform port exposed to custom HTTP modules.
// Available preserves the existing no-coordinator fallback behavior.
type IdempotencyCoordinator interface {
	Available() bool
	Execute(ctx context.Context, options IdempotencyOptions, execute func(context.Context) (any, error)) (*IdempotencyResult, error)
	IsStoreUnavailable(err error) bool
	RecordStoreUnavailable(route, scope, strategy string)
	RetryAfterSeconds(err error) int
}

// DefaultIdempotencyCoordinator resolves the current core coordinator for each
// call. This is intentional: existing handlers observe late initialization and
// test-time replacement through service.SetDefaultIdempotencyCoordinator.
func DefaultIdempotencyCoordinator() IdempotencyCoordinator {
	return defaultIdempotencyCoordinator{}
}

type defaultIdempotencyCoordinator struct{}

func (defaultIdempotencyCoordinator) Available() bool {
	return service.DefaultIdempotencyCoordinator() != nil
}

func (defaultIdempotencyCoordinator) Execute(
	ctx context.Context,
	options IdempotencyOptions,
	execute func(context.Context) (any, error),
) (*IdempotencyResult, error) {
	coordinator := service.DefaultIdempotencyCoordinator()
	if coordinator == nil {
		return nil, service.ErrIdempotencyStoreUnavail
	}
	result, err := coordinator.Execute(ctx, service.IdempotencyExecuteOptions{
		Scope: options.Scope, ActorScope: options.ActorScope, Method: options.Method, Route: options.Route,
		IdempotencyKey: options.IdempotencyKey, Payload: options.Payload, TTL: options.TTL, RequireKey: options.RequireKey,
	}, execute)
	if err != nil || result == nil {
		return nil, err
	}
	return &IdempotencyResult{Data: result.Data, Replayed: result.Replayed}, nil
}

func (defaultIdempotencyCoordinator) IsStoreUnavailable(err error) bool {
	return infraerrors.Code(err) == infraerrors.Code(service.ErrIdempotencyStoreUnavail)
}

func (defaultIdempotencyCoordinator) RecordStoreUnavailable(route, scope, strategy string) {
	service.RecordIdempotencyStoreUnavailable(route, scope, strategy)
}

func (defaultIdempotencyCoordinator) RetryAfterSeconds(err error) int {
	return service.RetryAfterSecondsFromError(err)
}

var _ IdempotencyCoordinator = defaultIdempotencyCoordinator{}

// NormalizeIdempotencyKey exposes the core key contract to custom modules
// without coupling their business services to the core service package.
func NormalizeIdempotencyKey(raw string) (string, error) {
	return service.NormalizeIdempotencyKey(raw)
}
