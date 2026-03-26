package service

import (
	"context"
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

// ScanBigKeysStream iterates keys matching pattern and streams progress
// via the onProgress callback. Respects maxScan limit and context cancellation.
// Element counts are fetched in a single pipeline at the end for top-N only.
func (s *AnalysisService) ScanBigKeysStream(
	ctx context.Context,
	pattern string,
	topN int,
	thresholdBytes int64,
	maxScan int64,
	onProgress func(api.ScanProgress),
) {
	start := time.Now()

	if topN <= 0 {
		topN = 20
	}
	if pattern == "" {
		pattern = "*"
	}
	if maxScan <= 0 {
		maxScan = 10000
	}

	var (
		cursor      uint64
		scannedKeys int64
		candidates  []api.BigKeyResult
		lastReport  time.Time
	)

	for scannedKeys < maxScan {
		if ctx.Err() != nil {
			break
		}

		batchSize := int64(200)
		if remaining := maxScan - scannedKeys; remaining < batchSize {
			batchSize = remaining
		}

		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			s.logger.Error("big key scan error", zap.Error(err))
			break
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
				candidates = append(candidates, api.BigKeyResult{
					Key:  key,
					Type: typeCmds[i].Val(),
					Size: size,
				})
			}
		}

		if time.Since(lastReport) > 300*time.Millisecond {
			onProgress(api.ScanProgress{
				ScannedKeys: scannedKeys,
				Found:       len(candidates),
				Elapsed:     time.Since(start).Truncate(time.Millisecond).String(),
			})
			lastReport = time.Now()
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Size > candidates[j].Size
	})
	if len(candidates) > topN {
		candidates = candidates[:topN]
	}

	s.fillElementCounts(ctx, candidates)

	onProgress(api.ScanProgress{
		ScannedKeys: scannedKeys,
		Found:       len(candidates),
		Elapsed:     time.Since(start).Truncate(time.Millisecond).String(),
		Done:        true,
		Results:     candidates,
		Duration:    time.Since(start).Truncate(time.Millisecond).String(),
	})
}

// fillElementCounts pipelines all element count commands in a single batch.
func (s *AnalysisService) fillElementCounts(ctx context.Context, results []api.BigKeyResult) {
	if len(results) == 0 {
		return
	}

	pipe := s.rdb.Pipeline()
	cmds := make([]*redis.IntCmd, len(results))
	for i, r := range results {
		switch r.Type {
		case "string":
			cmds[i] = pipe.StrLen(ctx, r.Key)
		case "hash":
			cmds[i] = pipe.HLen(ctx, r.Key)
		case "list":
			cmds[i] = pipe.LLen(ctx, r.Key)
		case "set":
			cmds[i] = pipe.SCard(ctx, r.Key)
		case "zset":
			cmds[i] = pipe.ZCard(ctx, r.Key)
		case "stream":
			cmds[i] = pipe.XLen(ctx, r.Key)
		}
	}
	_, _ = pipe.Exec(ctx)

	for i := range results {
		if cmds[i] != nil {
			results[i].ElementCount = cmds[i].Val()
		}
	}
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
