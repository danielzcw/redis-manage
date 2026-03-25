package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/danielwang/redis-manage/pkg/api"
)

type KeyService struct {
	rdb    *redis.Client
	parser *ParserService
	logger *zap.Logger
}

func NewKeyService(rdb *redis.Client, logger *zap.Logger) *KeyService {
	return &KeyService{
		rdb:    rdb,
		parser: NewParserService(),
		logger: logger,
	}
}

func (s *KeyService) ServerInfo(ctx context.Context) (*api.ServerInfo, error) {
	infoStr, err := s.rdb.Info(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("get server info: %w", err)
	}

	info := parseRedisInfo(infoStr)
	dbSize, _ := s.rdb.DBSize(ctx).Result()

	return &api.ServerInfo{
		Version:       info["redis_version"],
		Mode:          info["redis_mode"],
		OS:            info["os"],
		UsedMemory:    info["used_memory_human"],
		UsedMemoryRSS: info["used_memory_rss_human"],
		Clients:       info["connected_clients"],
		TotalKeys:     dbSize,
		UptimeSeconds: info["uptime_in_seconds"],
	}, nil
}

// ScanKeys uses SCAN with pipelined TYPE/TTL to avoid N+1 queries.
func (s *KeyService) ScanKeys(ctx context.Context, pattern string, cursor uint64, count int64) (*api.ScanKeysResponse, error) {
	keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, count).Result()
	if err != nil {
		return nil, fmt.Errorf("scan keys: %w", err)
	}

	if len(keys) == 0 {
		return &api.ScanKeysResponse{Keys: []api.KeyInfo{}, Cursor: nextCursor}, nil
	}

	pipe := s.rdb.Pipeline()
	typeCmds := make([]*redis.StatusCmd, len(keys))
	ttlCmds := make([]*redis.DurationCmd, len(keys))
	for i, key := range keys {
		typeCmds[i] = pipe.Type(ctx, key)
		ttlCmds[i] = pipe.TTL(ctx, key)
	}
	if _, err = pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	infos := make([]api.KeyInfo, 0, len(keys))
	for i, key := range keys {
		infos = append(infos, api.KeyInfo{
			Key:  key,
			Type: typeCmds[i].Val(),
			TTL:  ttlCmds[i].Val(),
		})
	}

	return &api.ScanKeysResponse{Keys: infos, Cursor: nextCursor}, nil
}

func (s *KeyService) GetKeyDetail(ctx context.Context, key string) (*api.KeyDetail, error) {
	keyType, err := s.rdb.Type(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("get key type: %w", err)
	}
	if keyType == "none" {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	ttl, _ := s.rdb.TTL(ctx, key).Result()
	memUsage, _ := s.rdb.MemoryUsage(ctx, key).Result()

	detail := &api.KeyDetail{
		Key:  key,
		Type: keyType,
		TTL:  int64(ttl.Seconds()),
		Size: memUsage,
	}

	switch keyType {
	case "string":
		raw, err := s.rdb.Get(ctx, key).Bytes()
		if err != nil {
			return nil, fmt.Errorf("get string value: %w", err)
		}
		parsed := s.parser.Parse(raw)
		detail.Value = parsed.Value
		detail.Format = parsed.Format
		detail.Length = int64(len(raw))

	case "hash":
		val, err := s.rdb.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("get hash value: %w", err)
		}
		detail.Value = val
		detail.Format = "hash"
		detail.Length, _ = s.rdb.HLen(ctx, key).Result()

	case "list":
		val, err := s.rdb.LRange(ctx, key, 0, 99).Result()
		if err != nil {
			return nil, fmt.Errorf("get list value: %w", err)
		}
		detail.Value = val
		detail.Format = "list"
		detail.Length, _ = s.rdb.LLen(ctx, key).Result()

	case "set":
		val, err := s.rdb.SMembers(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("get set value: %w", err)
		}
		detail.Value = val
		detail.Format = "set"
		detail.Length, _ = s.rdb.SCard(ctx, key).Result()

	case "zset":
		val, err := s.rdb.ZRangeWithScores(ctx, key, 0, 99).Result()
		if err != nil {
			return nil, fmt.Errorf("get zset value: %w", err)
		}
		detail.Value = val
		detail.Format = "zset"
		detail.Length, _ = s.rdb.ZCard(ctx, key).Result()

	case "stream":
		msgs, err := s.rdb.XRange(ctx, key, "-", "+").Result()
		if err != nil {
			return nil, fmt.Errorf("get stream value: %w", err)
		}
		if len(msgs) > 100 {
			msgs = msgs[:100]
		}
		detail.Value = msgs
		detail.Format = "stream"
		detail.Length, _ = s.rdb.XLen(ctx, key).Result()

	default:
		detail.Value = nil
		detail.Format = "unknown"
	}

	return detail, nil
}

func (s *KeyService) DeleteKey(ctx context.Context, key string) error {
	result, err := s.rdb.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("delete key: %w", err)
	}
	if result == 0 {
		return fmt.Errorf("key not found: %s", key)
	}
	return nil
}

func (s *KeyService) SetKeyTTL(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return s.rdb.Persist(ctx, key).Err()
	}
	return s.rdb.Expire(ctx, key, ttl).Err()
}

func (s *KeyService) SetStringValue(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.rdb.Set(ctx, key, value, ttl).Err()
}

func parseRedisInfo(info string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(info, "\r\n") {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
