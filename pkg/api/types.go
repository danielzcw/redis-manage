package api

import "time"

type KeyInfo struct {
	Key  string        `json:"key"`
	Type string        `json:"type"`
	TTL  time.Duration `json:"ttl"`
}

type KeyDetail struct {
	Key    string      `json:"key"`
	Type   string      `json:"type"`
	TTL    int64       `json:"ttl"`
	Size   int64       `json:"size"`
	Value  interface{} `json:"value"`
	Format string      `json:"format"`
	Length int64       `json:"length"`
}

type ScanKeysResponse struct {
	Keys   []KeyInfo `json:"keys"`
	Cursor uint64    `json:"cursor"`
}

type ServerInfo struct {
	Version       string `json:"version"`
	Mode          string `json:"mode"`
	OS            string `json:"os"`
	UsedMemory    string `json:"used_memory"`
	UsedMemoryRSS string `json:"used_memory_rss"`
	Clients       string `json:"connected_clients"`
	TotalKeys     int64  `json:"total_keys"`
	UptimeSeconds string `json:"uptime_in_seconds"`
}

type QueueInfo struct {
	Key    string `json:"key"`
	Type   string `json:"type"`
	Length int64  `json:"length"`
}

type QueueDetail struct {
	Key     string      `json:"key"`
	Type    string      `json:"type"`
	Length  int64       `json:"length"`
	Entries interface{} `json:"entries"`
	Groups  interface{} `json:"groups,omitempty"`
}

type StreamGroupInfo struct {
	Name            string `json:"name"`
	Consumers       int64  `json:"consumers"`
	Pending         int64  `json:"pending"`
	LastDeliveredID string `json:"last_delivered_id"`
}

type BigKeyResult struct {
	Key          string `json:"key"`
	Type         string `json:"type"`
	Size         int64  `json:"size"`
	ElementCount int64  `json:"element_count"`
}

type HotKeyResult struct {
	Key       string `json:"key"`
	Frequency int64  `json:"frequency"`
	Type      string `json:"type"`
}

type AnalysisResponse struct {
	Results     interface{} `json:"results"`
	ScannedKeys int64       `json:"scanned_keys"`
	Duration    string      `json:"duration"`
}

type ScanProgress struct {
	ScannedKeys int64          `json:"scanned_keys"`
	Found       int            `json:"found"`
	Elapsed     string         `json:"elapsed"`
	Done        bool           `json:"done"`
	Results     []BigKeyResult `json:"results,omitempty"`
	Duration    string         `json:"duration,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
