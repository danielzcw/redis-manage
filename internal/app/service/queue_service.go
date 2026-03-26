package service

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/danielwang/redis-manage/pkg/api"
)

type QueueService struct {
	rdb    *redis.Client
	logger *zap.Logger
}

func NewQueueService(rdb *redis.Client, logger *zap.Logger) *QueueService {
	return &QueueService{rdb: rdb, logger: logger}
}

// ListQueues scans all keys and filters for list/stream types.
func (s *QueueService) ListQueues(ctx context.Context, pattern string) ([]api.QueueInfo, error) {
	var queues []api.QueueInfo
	var cursor uint64

	for {
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan queues: %w", err)
		}

		if len(keys) > 0 {
			pipe := s.rdb.Pipeline()
			typeCmds := make([]*redis.StatusCmd, len(keys))
			for i, key := range keys {
				typeCmds[i] = pipe.Type(ctx, key)
			}
			if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
				return nil, fmt.Errorf("pipeline exec: %w", err)
			}

			for i, key := range keys {
				t := typeCmds[i].Val()
				if t != "list" && t != "stream" {
					continue
				}

				var length int64
				switch t {
				case "list":
					length, _ = s.rdb.LLen(ctx, key).Result()
				case "stream":
					length, _ = s.rdb.XLen(ctx, key).Result()
				}

				queues = append(queues, api.QueueInfo{
					Key:    key,
					Type:   t,
					Length: length,
				})
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if queues == nil {
		queues = []api.QueueInfo{}
	}
	return queues, nil
}

func (s *QueueService) GetQueueDetail(ctx context.Context, key string) (*api.QueueDetail, error) {
	keyType, err := s.rdb.Type(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("get key type: %w", err)
	}

	detail := &api.QueueDetail{Key: key, Type: keyType}

	switch keyType {
	case "list":
		detail.Length, _ = s.rdb.LLen(ctx, key).Result()
		entries, err := s.rdb.LRange(ctx, key, 0, 99).Result()
		if err != nil {
			return nil, fmt.Errorf("get list entries: %w", err)
		}
		detail.Entries = entries

	case "stream":
		detail.Length, _ = s.rdb.XLen(ctx, key).Result()
		msgs, err := s.rdb.XRange(ctx, key, "-", "+").Result()
		if err != nil {
			return nil, fmt.Errorf("get stream entries: %w", err)
		}
		if len(msgs) > 100 {
			msgs = msgs[:100]
		}
		detail.Entries = msgs

		groups, err := s.rdb.XInfoGroups(ctx, key).Result()
		if err == nil {
			groupInfos := make([]api.StreamGroupInfo, 0, len(groups))
			for _, g := range groups {
				groupInfos = append(groupInfos, api.StreamGroupInfo{
					Name:            g.Name,
					Consumers:       g.Consumers,
					Pending:         g.Pending,
					LastDeliveredID: g.LastDeliveredID,
				})
			}
			detail.Groups = groupInfos
		}

	default:
		return nil, fmt.Errorf("key %s is not a queue type (got %s)", key, keyType)
	}

	return detail, nil
}

func (s *QueueService) Push(ctx context.Context, key string, values []string) error {
	keyType, err := s.rdb.Type(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("get key type: %w", err)
	}

	switch keyType {
	case "list", "none":
		args := make([]interface{}, len(values))
		for i, v := range values {
			args[i] = v
		}
		return s.rdb.RPush(ctx, key, args...).Err()

	case "stream":
		for _, v := range values {
			if err := s.rdb.XAdd(ctx, &redis.XAddArgs{
				Stream: key,
				Values: map[string]interface{}{"data": v},
			}).Err(); err != nil {
				return fmt.Errorf("xadd: %w", err)
			}
		}
		return nil

	default:
		return fmt.Errorf("key %s is not a queue type (got %s)", key, keyType)
	}
}

func (s *QueueService) Pop(ctx context.Context, key string, count int64) ([]string, error) {
	keyType, err := s.rdb.Type(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("get key type: %w", err)
	}

	if keyType != "list" {
		return nil, fmt.Errorf("pop only supported for list type (got %s)", keyType)
	}

	var results []string
	for i := int64(0); i < count; i++ {
		val, err := s.rdb.LPop(ctx, key).Result()
		if err == redis.Nil {
			break
		}
		if err != nil {
			return results, fmt.Errorf("lpop: %w", err)
		}
		results = append(results, val)
	}

	return results, nil
}
