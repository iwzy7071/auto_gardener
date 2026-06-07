package app

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const pricingNote = "仅统计底层 CLI token 消耗和使用模型；界面不展示费用估算。"

type usageLogEvent struct {
	Time       time.Time `json:"time"`
	TaskID     string    `json:"taskId"`
	RunID      string    `json:"runId"`
	SourceType string    `json:"sourceType"`
	SourceID   string    `json:"sourceId,omitempty"`
	SourceName string    `json:"sourceName"`
	Line       string    `json:"line"`
}

type usageStreamState struct {
	runID             string
	externalRunID     bool
	sourceType        string
	sourceID          string
	sourceName        string
	model             string
	expectTotalTokens bool
	inputTokens       int64
	cachedInputTokens int64
	outputTokens      int64
}

type usageRecorder struct {
	mu          sync.Mutex
	store       *Store
	taskID      string
	runID       string
	sourceType  string
	sourceID    string
	sourceName  string
	expectTotal bool
}

func (o *Orchestrator) newUsageRecorder(taskID, runID, sourceType, sourceID, sourceName string) *usageRecorder {
	return &usageRecorder{
		store:      o.store,
		taskID:     taskID,
		runID:      runID,
		sourceType: sourceType,
		sourceID:   sourceID,
		sourceName: sourceName,
	}
}

func (r *usageRecorder) Record(raw string) {
	cleaned := cleanCodexUsageLine(raw)
	if cleaned == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	lower := strings.ToLower(cleaned)
	shouldWrite := isModelUsageLine(cleaned) || isTokenDetailLine(cleaned)
	if lower == "tokens used" || strings.HasPrefix(lower, "tokens used") {
		r.expectTotal = true
		shouldWrite = true
	} else if r.expectTotal && isMostlyNumber(cleaned) {
		r.expectTotal = false
		shouldWrite = true
	}
	if shouldWrite {
		r.store.AppendUsageEvent(r.taskID, usageLogEvent{
			Time:       time.Now(),
			TaskID:     r.taskID,
			RunID:      r.runID,
			SourceType: r.sourceType,
			SourceID:   r.sourceID,
			SourceName: r.sourceName,
			Line:       cleaned,
		})
	}
}

func (s *Store) AppendUsageEvent(taskID string, event usageLogEvent) {
	if strings.TrimSpace(event.Line) == "" {
		return
	}
	path := filepath.Join(s.dataDir, "forests", taskID, "usage.jsonl")
	b, err := json.Marshal(event)
	if err != nil {
		return
	}
	_ = appendFile(path, string(b)+"\n")
	if s.events != nil {
		if task, ok := s.GetTask(taskID); ok {
			s.events.Publish(taskID, task)
		}
	}
}

func (s *Store) TaskUsage(taskID string) (TokenUsageSummary, error) {
	task, ok := s.GetTask(taskID)
	if !ok {
		return TokenUsageSummary{}, ErrNotFound
	}
	records := make([]TokenUsageRecord, 0)
	records = append(records, parseUsageJSONL(filepath.Join(s.forestDir(taskID), "usage.jsonl"), taskID)...)
	records = append(records, parseLegacyUsage(task, s.forestDir(taskID))...)
	records = dedupeUsageRecords(records)
	return summarizeUsage(taskID, records), nil
}

func (s *Store) AllUsage() []TokenUsageSummary {
	tasks := s.ListTasks()
	out := make([]TokenUsageSummary, 0, len(tasks))
	for _, task := range tasks {
		summary, err := s.TaskUsage(task.ID)
		if err == nil {
			summary.Records = nil
			out = append(out, summary)
		}
	}
	return out
}

func parseUsageJSONL(path, taskID string) []TokenUsageRecord {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	states := map[string]*usageStreamState{}
	var records []TokenUsageRecord
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		var ev usageLogEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		if strings.TrimSpace(ev.TaskID) != "" && ev.TaskID != taskID {
			continue
		}
		ev.TaskID = taskID
		key := ev.RunID
		if key == "" {
			key = ev.SourceType + ":" + ev.SourceID
		}
		st := states[key]
		if st == nil {
			st = &usageStreamState{runID: ev.RunID, externalRunID: ev.RunID != "", sourceType: ev.SourceType, sourceID: ev.SourceID, sourceName: ev.SourceName}
			states[key] = st
		}
		if rec, ok := consumeUsageLine(st, ev.Line, ev.Time, ev.TaskID); ok {
			records = append(records, rec)
		}
	}
	return records
}

func parseLegacyUsage(task *Task, forestDir string) []TokenUsageRecord {
	var records []TokenUsageRecord
	if task.LogPath != "" {
		records = append(records, parseLegacyUsageFile(task.ID, task.LogPath, "gardener", "", "Gardener")...)
	}
	for _, tr := range task.Trees {
		if tr == nil {
			continue
		}
		path := filepath.Join(forestDir, "trees", tr.ID, "progress.log")
		name := tr.Name
		if name == "" {
			name = "Tree"
		}
		records = append(records, parseLegacyUsageFile(task.ID, path, "tree", tr.ID, name)...)
	}
	return records
}

func parseLegacyUsageFile(taskID, path, sourceType, sourceID, sourceName string) []TokenUsageRecord {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	st := &usageStreamState{sourceType: sourceType, sourceID: sourceID, sourceName: sourceName}
	var records []TokenUsageRecord
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		ts, payload, ok := splitTimestampedLine(line)
		if !ok {
			continue
		}
		cleaned := cleanLegacyUsagePayload(payload)
		if cleaned == "" {
			continue
		}
		if rec, ok := consumeUsageLine(st, cleaned, ts, taskID); ok {
			records = append(records, rec)
		}
	}
	return records
}

func consumeUsageLine(st *usageStreamState, line string, when time.Time, taskID string) (TokenUsageRecord, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return TokenUsageRecord{}, false
	}
	if model, ok := parseModelLine(line); ok {
		st.model = model
		if !st.externalRunID || st.runID == "" {
			st.runID = newStableUsageID(taskID, st.sourceType, st.sourceID, st.sourceName, when.Format(time.RFC3339Nano), model)
		}
		st.expectTotalTokens = false
		st.inputTokens, st.cachedInputTokens, st.outputTokens = 0, 0, 0
		return TokenUsageRecord{}, false
	}
	if kind, n, ok := parseTokenDetailLine(line); ok {
		switch kind {
		case "input":
			st.inputTokens = n
		case "cached_input":
			st.cachedInputTokens = n
		case "output":
			st.outputTokens = n
		case "total":
			return buildUsageRecord(st, taskID, n, when), true
		}
		return TokenUsageRecord{}, false
	}
	lower := strings.ToLower(line)
	if lower == "tokens used" || strings.HasPrefix(lower, "tokens used") {
		st.expectTotalTokens = true
		return TokenUsageRecord{}, false
	}
	if st.expectTotalTokens && isMostlyNumber(line) {
		n := parseNumber(line)
		st.expectTotalTokens = false
		return buildUsageRecord(st, taskID, n, when), true
	}
	return TokenUsageRecord{}, false
}

func buildUsageRecord(st *usageStreamState, taskID string, total int64, when time.Time) TokenUsageRecord {
	model := st.model
	if model == "" {
		model = "unknown"
	}
	if total <= 0 && (st.inputTokens > 0 || st.cachedInputTokens > 0 || st.outputTokens > 0) {
		total = st.inputTokens + st.cachedInputTokens + st.outputTokens
	}
	runID := st.runID
	if runID == "" {
		runID = newStableUsageID(taskID, st.sourceType, st.sourceID, st.sourceName, when.Format(time.RFC3339Nano), model, strconv.FormatInt(total, 10))
	}
	rec := TokenUsageRecord{
		ID:                newStableUsageID(taskID, runID, st.sourceType, st.sourceID, model, strconv.FormatInt(total, 10), when.Format(time.RFC3339Nano)),
		TaskID:            taskID,
		RunID:             runID,
		SourceType:        firstNonEmpty(st.sourceType, "agent"),
		SourceID:          st.sourceID,
		SourceName:        firstNonEmpty(st.sourceName, st.sourceType),
		Model:             model,
		TotalTokens:       total,
		InputTokens:       st.inputTokens,
		CachedInputTokens: st.cachedInputTokens,
		OutputTokens:      st.outputTokens,
		CreatedAt:         when,
	}
	applyPrice(&rec)
	st.inputTokens, st.cachedInputTokens, st.outputTokens = 0, 0, 0
	return rec
}

func dedupeUsageRecords(records []TokenUsageRecord) []TokenUsageRecord {
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})
	seen := map[string]bool{}
	out := make([]TokenUsageRecord, 0, len(records))
	for _, rec := range records {
		if rec.TotalTokens <= 0 {
			continue
		}
		key := rec.ID
		if rec.RunID != "" {
			key = rec.TaskID + "|" + rec.RunID + "|" + rec.Model + "|" + strconv.FormatInt(rec.TotalTokens, 10)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, rec)
	}
	return out
}

func summarizeUsage(taskID string, records []TokenUsageRecord) TokenUsageSummary {
	summary := TokenUsageSummary{TaskID: taskID, Records: records, PricingNote: pricingNote, ExactCost: true}
	byModel := map[string]*TokenUsageModelSummary{}
	for _, rec := range records {
		summary.TotalTokens += rec.TotalTokens
		summary.CostUSD += rec.CostUSD
		summary.MinCostUSD += rec.MinCostUSD
		summary.MaxCostUSD += rec.MaxCostUSD
		if rec.Priced {
			summary.Priced = true
		}
		if !rec.ExactCost {
			summary.ExactCost = false
		}
		if summary.LastUpdatedAt == nil || rec.CreatedAt.After(*summary.LastUpdatedAt) {
			t := rec.CreatedAt
			summary.LastUpdatedAt = &t
		}
		model := rec.Model
		if model == "" {
			model = "unknown"
		}
		ms := byModel[model]
		if ms == nil {
			ms = &TokenUsageModelSummary{Model: model, ExactCost: true}
			byModel[model] = ms
		}
		ms.TotalTokens += rec.TotalTokens
		ms.CostUSD += rec.CostUSD
		ms.MinCostUSD += rec.MinCostUSD
		ms.MaxCostUSD += rec.MaxCostUSD
		ms.Runs++
		if rec.Priced {
			ms.Priced = true
		}
		if !rec.ExactCost {
			ms.ExactCost = false
		}
	}
	for _, ms := range byModel {
		summary.Models = append(summary.Models, *ms)
	}
	sort.Slice(summary.Models, func(i, j int) bool {
		return summary.Models[i].TotalTokens > summary.Models[j].TotalTokens
	})
	if len(records) == 0 {
		summary.ExactCost = false
	}
	return summary
}

func applyPrice(rec *TokenUsageRecord) {
	// Product decision: Gardener currently exposes token consumption only.
	// Keep the historical JSON fields zero-valued for API compatibility, but do
	// not calculate or display spend estimates.
	rec.Priced = false
	rec.ExactCost = false
}

func modelPrices() map[string]ModelPrice {
	prices := map[string]ModelPrice{
		// Defaults mirror current OpenAI API style pricing fields. Unknown or
		// custom Codex models can be supplied through AUTO_GARDENER_MODEL_PRICES_JSON.
		"gpt-5.5":      {InputPerMTok: 5.00, CachedInputPerMTok: 0.50, OutputPerMTok: 30.00},
		"gpt-5.4":      {InputPerMTok: 2.50, CachedInputPerMTok: 0.25, OutputPerMTok: 15.00},
		"gpt-5":        {InputPerMTok: 1.25, CachedInputPerMTok: 0.125, OutputPerMTok: 10.00},
		"gpt-5-mini":   {InputPerMTok: 0.25, CachedInputPerMTok: 0.025, OutputPerMTok: 2.00},
		"gpt-5-nano":   {InputPerMTok: 0.05, CachedInputPerMTok: 0.005, OutputPerMTok: 0.40},
		"gpt-5.4-mini": {InputPerMTok: 0.75, CachedInputPerMTok: 0.075, OutputPerMTok: 4.50},
	}
	raw := strings.TrimSpace(os.Getenv("AUTO_GARDENER_MODEL_PRICES_JSON"))
	if raw == "" {
		return prices
	}
	var overrides map[string]ModelPrice
	if err := json.Unmarshal([]byte(raw), &overrides); err != nil {
		return prices
	}
	for model, price := range overrides {
		model = strings.ToLower(strings.TrimSpace(model))
		if model == "" {
			continue
		}
		prices[model] = price
	}
	return prices
}

func splitTimestampedLine(line string) (time.Time, string, bool) {
	if !strings.HasPrefix(line, "[") {
		return time.Time{}, "", false
	}
	end := strings.Index(line, "]")
	if end <= 1 {
		return time.Time{}, "", false
	}
	ts, err := time.Parse(time.RFC3339, line[1:end])
	if err != nil {
		return time.Time{}, "", false
	}
	return ts, strings.TrimSpace(line[end+1:]), true
}

func cleanLegacyUsagePayload(payload string) string {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return ""
	}
	idx := strings.Index(payload, "stderr:")
	if idx < 0 {
		return ""
	}
	cleaned := strings.TrimSpace(payload[idx+len("stderr:"):])
	if strings.HasPrefix(cleaned, "[20") {
		return ""
	}
	return cleaned
}

func cleanCodexUsageLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "stderr:")
	return strings.TrimSpace(line)
}

func isModelUsageLine(line string) bool {
	_, ok := parseModelLine(line)
	return ok
}

func parseModelLine(line string) (string, bool) {
	m := regexp.MustCompile(`(?i)^model\s*:\s*([A-Za-z0-9._:-]+)`).FindStringSubmatch(strings.TrimSpace(line))
	if len(m) != 2 {
		return "", false
	}
	return strings.TrimSpace(m[1]), true
}

func isTokenDetailLine(line string) bool {
	_, _, ok := parseTokenDetailLine(line)
	return ok
}

func parseTokenDetailLine(line string) (string, int64, bool) {
	trimmed := strings.TrimSpace(line)
	lower := strings.ToLower(trimmed)
	patterns := []struct {
		kind string
		re   *regexp.Regexp
	}{
		{"cached_input", regexp.MustCompile(`(?i)^(cached\s+input|input\s+cached)\s+tokens?\s*[:=]\s*([0-9][0-9,.\s]*)`)},
		{"input", regexp.MustCompile(`(?i)^input\s+tokens?\s*[:=]\s*([0-9][0-9,.\s]*)`)},
		{"output", regexp.MustCompile(`(?i)^output\s+tokens?\s*[:=]\s*([0-9][0-9,.\s]*)`)},
		{"total", regexp.MustCompile(`(?i)^(total\s+tokens?|tokens\s+used)\s*[:=]\s*([0-9][0-9,.\s]*)`)},
	}
	for _, p := range patterns {
		m := p.re.FindStringSubmatch(trimmed)
		if len(m) == 0 {
			continue
		}
		num := m[len(m)-1]
		if p.kind == "cached_input" && !strings.Contains(lower, "token") {
			continue
		}
		return p.kind, parseNumber(num), true
	}
	return "", 0, false
}

func parseNumber(s string) int64 {
	cleaned := strings.NewReplacer(",", "", " ", "", "\t", "").Replace(strings.TrimSpace(s))
	if strings.Contains(cleaned, ".") {
		f, _ := strconv.ParseFloat(cleaned, 64)
		return int64(f)
	}
	n, _ := strconv.ParseInt(cleaned, 10, 64)
	return n
}

func newStableUsageID(parts ...string) string {
	h := sha1.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return "usage_" + hex.EncodeToString(h.Sum(nil))[:16]
}

func firstNonEmpty(items ...string) string {
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			return item
		}
	}
	return ""
}

func minPositive(a, b float64) float64 {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (r TokenUsageRecord) String() string {
	return fmt.Sprintf("%s %s %d", r.SourceName, r.Model, r.TotalTokens)
}
