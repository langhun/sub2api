package smartrouting

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type responseRoute struct {
	GroupID   int64 `json:"group_id"`
	AccountID int64 `json:"account_id"`
}

func responseRouteKey(key *service.APIKey, primaryID int64, responseID string) string {
	digest := sha256.Sum256([]byte(responseID))
	return fmt.Sprintf("custom:route:response:%d:%d:%d:%x", key.UserID, key.ID, primaryID, digest)
}

func (m *Module) BindRoutingResponse(ctx context.Context, key *service.APIKey, primaryID, groupID, accountID int64, responseID string, ttl time.Duration) error {
	if m.redis == nil {
		return errors.New("routing response store unavailable")
	}
	if key == nil || primaryID <= 0 || groupID <= 0 || accountID <= 0 || responseID == "" {
		return errors.New("invalid routing response owner")
	}
	body, err := json.Marshal(responseRoute{groupID, accountID})
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	return m.redis.Set(writeCtx, responseRouteKey(key, primaryID, responseID), body, ttl).Err()
}

func (m *Module) LookupRoutingResponse(ctx context.Context, key *service.APIKey, responseID string) (int64, int64, error) {
	if key == nil || key.GroupID == nil {
		return 0, 0, nil
	}
	p, err := m.repo.Get(ctx, key.ID)
	if err != nil {
		// A profile-store outage must not break ordinary single-group response
		// chains. A known routed response, however, cannot safely lose its owner.
		if m.redis != nil {
			readCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()
			if exists, readErr := m.redis.Exists(readCtx, responseRouteKey(key, *key.GroupID, responseID)).Result(); readErr == nil && exists > 0 {
				return 0, 0, err
			}
		}
		return 0, 0, nil
	}
	if !routingProfileUsable(p) || !p.Enabled || p.Mode != "smart" {
		if m.redis != nil {
			readCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()
			body, readErr := m.redis.Get(readCtx, responseRouteKey(key, *key.GroupID, responseID)).Bytes()
			var previous responseRoute
			if readErr == nil && json.Unmarshal(body, &previous) == nil && previous.GroupID != *key.GroupID {
				return 0, 0, errors.New("response candidate route is no longer enabled")
			}
		}
		return 0, 0, nil
	}
	if m.redis == nil {
		return 0, 0, errors.New("routing response store unavailable")
	}
	readCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	body, err := m.redis.Get(readCtx, responseRouteKey(key, *key.GroupID, responseID)).Bytes()
	var route responseRoute
	if errors.Is(err, redis.Nil) {
		// Rolling upgrades may still have responses recorded before the Overlay
		// owner index existed. Recover only from authenticated account bindings,
		// never from the primary's owner-only alias or from a guessed account.
		if m.openAI != nil {
			ids := []int64{*key.GroupID}
			for _, g := range p.Groups {
				if g.Enabled && g.GroupID != *key.GroupID {
					ids = append(ids, g.GroupID)
				}
			}
			for _, id := range ids {
				accountID, lookupErr := m.openAI.RoutingResponseAccount(readCtx, id, responseID, key.UserID, key.ID)
				if lookupErr != nil {
					return 0, 0, lookupErr
				}
				if accountID > 0 {
					if route.AccountID > 0 {
						return 0, 0, errors.New("ambiguous response routing owner")
					}
					route = responseRoute{id, accountID}
				}
			}
		}
		if route.AccountID == 0 {
			return 0, 0, errors.New("response routing owner expired or unavailable")
		}
	} else if err != nil {
		return 0, 0, err
	} else if err := json.Unmarshal(body, &route); err != nil {
		return 0, 0, err
	}
	configured := route.GroupID == *key.GroupID
	for _, g := range p.Groups {
		if g.Enabled && g.GroupID == route.GroupID {
			configured = true
		}
	}
	if !configured || route.AccountID <= 0 {
		return 0, 0, errors.New("response route is no longer configured")
	}
	primary, err := m.groups.GetByIDLite(ctx, *key.GroupID)
	if err != nil {
		return 0, 0, err
	}
	group, err := m.groups.GetByIDLite(ctx, route.GroupID)
	if err != nil {
		return 0, 0, err
	}
	if !service.SmartRoutingGroupsCompatible(primary, group) || !service.SmartRoutingPrimaryContractMatches(key.Group, primary, *key.GroupID) {
		return 0, 0, errors.New("response route contract changed")
	}
	if route.GroupID != *key.GroupID {
		if m.availableGroups == nil {
			return 0, 0, errors.New("response route permissions unavailable")
		}
		groups, err := m.availableGroups(ctx, key.UserID)
		if err != nil {
			return 0, 0, err
		}
		allowed := false
		for _, g := range groups {
			if g.ID == route.GroupID {
				allowed = true
			}
		}
		if !allowed {
			return 0, 0, errors.New("response route permission revoked")
		}
	}
	return route.GroupID, route.AccountID, nil
}
