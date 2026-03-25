package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/danielwang/redis-manage/pkg/api"
)

type AnalysisService struct {
	rdb    *redis.Client
	logger *zap.Logger
}

func NewAnalysisService(rdb *redis.Client, logger *zap.Logger) *AnalysisService {
	return &AnalysisService{rdb: rdb, logger: logger}
}

// ScanBigKeys iterates all keys matching the pattern and returns the top N
// by memory usage. Uses pipelined MEMORY USAGE for efficiency.
func (s *AnalysisService) ScanBigKeys(ctx context.Context, pattern string, topN int, thresholdBytes int64) (*api.AnalysisResponse, error) {
	start := time.Now()

	if topN <= 0 {
		topN = 20
	}
	if pattern == "" {
		pattern = "*"
	}

	var (
		cursor      uint64
		scannedKeys int64
		results     []api.BigKeyResult
	)

	for {
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		if len(keys) > 0 {
			pipe := s.rdb.Pipeline()
			typeCmds := make([]*redis.StatusCmd, len(keys))
			memCmds := make([]*redis.IntCmd, len(keys))
			for i, key := range keys {
				typeCmds[i] = pipe.Type(ctx, key)
				memCmds[i] = pipe.MemoryUsage(ctx, key)
			}
			_, _ = pipe.Exec(ctx)

			for i, key := range keys {
				scannedKeys++
				size := memCmds[i].Val()
				if size < thresholdBytes {
					continue
				}

				keyType := typeCmds[i].Val()
				elemCount := s.getElementCount(ctx, key, keyType)

				results = append(results, api.BigKeyResult{
					Key:          key,
					Type:         keyType,
					Size:         size,
					ElementCount: elemCount,
				})
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
		if ctx.Err() != nil {
			break
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Size > results[j].Size
	})
	if len(results) > topN {
		results = results[:topN]
	}

	return &api.AnalysisResponse{
		Results:     results,
		ScannedKeys: scannedKeys,
		Duration:    time.Since(start).String(),
	}, nil
}

func (s *AnalysisService) getElementCount(ctx context.Context, key, keyType string) int64 {
	switch keyType {
	case "string":
		n, _ := s.rdb.StrLen(ctx, key).Result()
		return n
	case "hash":
		n, _ := s.rdb.HLen(ctx, key).Result()
		return n
	case "list":
		n, _ := s.rdb.LLen(ctx, key).Result()
		return n
	case "set":
		n, _ := s.rdb.SCard(ctx, key).Result()
		return n
	case "zset":
		n, _ := s.rdb.ZCard(ctx, key).Result()
		return n
	case "stream":
		n, _ := s.rdb.XLen(ctx, key).Result()
		return n
	}
	return 0
}

// ScanHotKeys samples random keys and uses OBJECT FREQ to estimate
// access frequency. Falls back to random sampling if LFU is not enabled.
func (s *AnalysisService) ScanHotKeys(ctx context.Context, sampleCount int) (*api.AnalysisResponse, error) {
	start := time.Now()

	if sampleCount <= 0 {
		sampleCount = 20
	}

	type keyFreq struct {
		key  string
		freq int64
	}

	freq := make(map[string]int64)
	lfu := true

	for i := 0; i < sampleCount*10; i++ {
		key, err := s.rdb.RandomKey(ctx).Result()
		if err != nil {
			break
		}

		if lfu {
			result, err := s.rdb.Do(ctx, "OBJECT", "FREQ", key).Int64()
			if err != nil {
				lfu = false
				freq[key]++
				continue
			}
			if result > freq[key] {
				freq[key] = result
			}
		} else {
			freq[key]++
		}
	}

	sorted := make([]keyFreq, 0, len(freq))
	for k, v := range freq {
		sorted = append(sorted, keyFreq{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].freq > sorted[j].freq
	})
	if len(sorted) > sampleCount {
		sorted = sorted[:sampleCount]
	}

	results := make([]api.HotKeyResult, 0, len(sorted))
	if len(sorted) > 0 {
		pipe := s.rdb.Pipeline()
		typeCmds := make([]*redis.StatusCmd, len(sorted))
		for i, kf := range sorted {
			typeCmds[i] = pipe.Type(ctx, kf.key)
		}
		_, _ = pipe.Exec(ctx)

		for i, kf := range sorted {
			results = append(results, api.HotKeyResult{
				Key:       kf.key,
				Frequency: kf.freq,
				Type:      typeCmds[i].Val(),
			})
		}
	}

	return &api.AnalysisResponse{
		Results:     results,
		ScannedKeys: int64(len(freq)),
		Duration:    time.Since(start).String(),
	}, nil
}
