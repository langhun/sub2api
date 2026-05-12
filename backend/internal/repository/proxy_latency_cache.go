package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const proxyLatencyKeyPrefix = "proxy:latency:"
const proxyLatencyMergeRetries = 5

func proxyLatencyKey(proxyID int64) string {
	return fmt.Sprintf("%s%d", proxyLatencyKeyPrefix, proxyID)
}

type proxyLatencyCache struct {
	rdb *redis.Client
}

func NewProxyLatencyCache(rdb *redis.Client) service.ProxyLatencyCache {
	return &proxyLatencyCache{rdb: rdb}
}

func (c *proxyLatencyCache) GetProxyLatencies(ctx context.Context, proxyIDs []int64) (map[int64]*service.ProxyLatencyInfo, error) {
	results := make(map[int64]*service.ProxyLatencyInfo)
	if len(proxyIDs) == 0 {
		return results, nil
	}

	uniqueIDs := make([]int64, 0, len(proxyIDs))
	seen := make(map[int64]struct{}, len(proxyIDs))
	for _, proxyID := range proxyIDs {
		if proxyID <= 0 {
			continue
		}
		if _, ok := seen[proxyID]; ok {
			continue
		}
		seen[proxyID] = struct{}{}
		uniqueIDs = append(uniqueIDs, proxyID)
	}
	if len(uniqueIDs) == 0 {
		return results, nil
	}

	type latencyReadCommand struct {
		proxyID int64
		key     string
		keyType string
		hash    *redis.MapStringStringCmd
		string  *redis.StringCmd
	}
	commands := make([]latencyReadCommand, 0, len(uniqueIDs))
	typeCommands := make([]*redis.StatusCmd, len(uniqueIDs))
	_, err := c.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for i, proxyID := range uniqueIDs {
			typeCommands[i] = pipe.Type(ctx, proxyLatencyKey(proxyID))
		}
		return nil
	})
	if err != nil {
		return results, err
	}

	_, err = c.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for i, proxyID := range uniqueIDs {
			key := proxyLatencyKey(proxyID)
			keyType, err := typeCommands[i].Result()
			if err != nil {
				return err
			}
			cmd := latencyReadCommand{proxyID: proxyID, key: key, keyType: keyType}
			switch keyType {
			case "hash":
				cmd.hash = pipe.HGetAll(ctx, key)
			case "string":
				cmd.string = pipe.Get(ctx, key)
			}
			commands = append(commands, cmd)
		}
		return nil
	})
	if err != nil && !errors.Is(err, redis.Nil) {
		return results, err
	}

	for _, cmd := range commands {
		var info *service.ProxyLatencyInfo
		var err error
		switch cmd.keyType {
		case "hash":
			fields, resultErr := cmd.hash.Result()
			if resultErr != nil {
				info, err = c.readProxyLatency(ctx, cmd.key)
				break
			}
			if len(fields) > 0 {
				info, err = proxyLatencyInfoFromHash(fields)
			}
		case "string":
			payload, resultErr := cmd.string.Bytes()
			if errors.Is(resultErr, redis.Nil) {
				continue
			}
			if resultErr != nil {
				info, err = c.readProxyLatency(ctx, cmd.key)
				break
			}
			var parsed service.ProxyLatencyInfo
			if json.Unmarshal(payload, &parsed) == nil {
				info = &parsed
			}
		default:
			continue
		}
		if err != nil {
			return results, err
		}
		if info != nil {
			results[cmd.proxyID] = info
		}
	}

	return results, nil
}

func (c *proxyLatencyCache) SetProxyLatency(ctx context.Context, proxyID int64, info *service.ProxyLatencyInfo) error {
	if info == nil {
		return nil
	}
	fields, err := proxyLatencyInfoToHash(info)
	if err != nil {
		return err
	}
	if len(fields) == 0 {
		return c.rdb.Del(ctx, proxyLatencyKey(proxyID)).Err()
	}
	_, err = c.rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		key := proxyLatencyKey(proxyID)
		pipe.Del(ctx, key)
		pipe.HSet(ctx, key, fields)
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *proxyLatencyCache) MergeProxyLatency(
	ctx context.Context,
	proxyID int64,
	apply func(*service.ProxyLatencyInfo),
) (*service.ProxyLatencyInfo, error) {
	if apply == nil {
		return nil, nil
	}

	key := proxyLatencyKey(proxyID)
	var merged *service.ProxyLatencyInfo
	for attempt := 0; attempt < proxyLatencyMergeRetries; attempt++ {
		err := c.rdb.Watch(ctx, func(tx *redis.Tx) error {
			existing, err := c.readProxyLatencyWithClient(ctx, tx, key)
			if err != nil {
				return err
			}

			info := &service.ProxyLatencyInfo{UpdatedAt: time.Now()}
			if existing != nil {
				cloned := *existing
				info = &cloned
			}
			apply(info)

			fields, err := proxyLatencyInfoToHash(info)
			if err != nil {
				return err
			}

			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(ctx, key)
				if len(fields) > 0 {
					pipe.HSet(ctx, key, fields)
				}
				return nil
			})
			if err != nil {
				return err
			}

			cloned := *info
			merged = &cloned
			return nil
		}, key)
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		return merged, err
	}
	return nil, redis.TxFailedErr
}

func (c *proxyLatencyCache) readProxyLatency(ctx context.Context, key string) (*service.ProxyLatencyInfo, error) {
	return c.readProxyLatencyWithClient(ctx, c.rdb, key)
}

func (c *proxyLatencyCache) readProxyLatencyWithClient(ctx context.Context, client redis.Cmdable, key string) (*service.ProxyLatencyInfo, error) {
	keyType, err := client.Type(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	switch keyType {
	case "none":
		return nil, nil
	case "hash":
		fields, err := client.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		if len(fields) == 0 {
			return nil, nil
		}
		return proxyLatencyInfoFromHash(fields)
	case "string":
		payload, err := client.Get(ctx, key).Bytes()
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		var info service.ProxyLatencyInfo
		if err := json.Unmarshal(payload, &info); err != nil {
			return nil, nil
		}
		return &info, nil
	default:
		return nil, nil
	}
}

func proxyLatencyInfoToHash(info *service.ProxyLatencyInfo) (map[string]any, error) {
	if info == nil {
		return nil, nil
	}
	payload, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}

	rawFields := make(map[string]json.RawMessage)
	if err := json.Unmarshal(payload, &rawFields); err != nil {
		return nil, err
	}

	fields := make(map[string]any, len(rawFields))
	for field, raw := range rawFields {
		fields[field] = string(raw)
	}
	return fields, nil
}

func proxyLatencyInfoFromHash(fields map[string]string) (*service.ProxyLatencyInfo, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	rawFields := make(map[string]json.RawMessage, len(fields))
	for field, value := range fields {
		raw := json.RawMessage(value)
		if !json.Valid(raw) {
			continue
		}
		rawFields[field] = raw
	}
	if len(rawFields) == 0 {
		return nil, nil
	}

	payload, err := json.Marshal(rawFields)
	if err != nil {
		return nil, err
	}

	var info service.ProxyLatencyInfo
	if err := json.Unmarshal(payload, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
