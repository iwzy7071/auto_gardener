package app

import "time"

type Status string

const (
	StatusRunning  Status = "Running"
	StatusFinished Status = "Finished"
)

type LogLevel string

const (
	LogLevelQuiet    LogLevel = "quiet"
	LogLevelNormal   LogLevel = "normal"
	LogLevelDetailed LogLevel = "detailed"
)

type ModelMode string

const (
	ModelModeDefault ModelMode = "default"
	ModelModeMiniMax ModelMode = "MiniMax-M3"
	ModelModeKimi    ModelMode = "kimi-k2.7-code"
)

type CLIEngine string

const (
	CLIEngineCodex  CLIEngine = "codex"
	CLIEngineClaude CLIEngine = "claude"
)

type AppSettings struct {
	LogLevel               LogLevel  `json:"logLevel"`
	ModelMode              ModelMode `json:"modelMode"`
	CLIEngine              CLIEngine `json:"cliEngine"`
	MiniMaxToken           string    `json:"minimaxToken,omitempty"`
	KimiToken              string    `json:"kimiToken,omitempty"`
	MiniMaxTokenConfigured bool      `json:"minimaxTokenConfigured,omitempty"`
	KimiTokenConfigured    bool      `json:"kimiTokenConfigured,omitempty"`
}

type ModelPrice struct {
	InputPerMTok       float64 `json:"inputPerMTok"`
	CachedInputPerMTok float64 `json:"cachedInputPerMTok"`
	OutputPerMTok      float64 `json:"outputPerMTok"`
}

type TokenUsageRecord struct {
	ID                string    `json:"id"`
	TaskID            string    `json:"taskId"`
	RunID             string    `json:"runId"`
	SourceType        string    `json:"sourceType"`
	SourceID          string    `json:"sourceId,omitempty"`
	SourceName        string    `json:"sourceName"`
	Model             string    `json:"model"`
	TotalTokens       int64     `json:"totalTokens"`
	InputTokens       int64     `json:"inputTokens,omitempty"`
	CachedInputTokens int64     `json:"cachedInputTokens,omitempty"`
	OutputTokens      int64     `json:"outputTokens,omitempty"`
	CostUSD           float64   `json:"costUSD,omitempty"`
	MinCostUSD        float64   `json:"minCostUSD,omitempty"`
	MaxCostUSD        float64   `json:"maxCostUSD,omitempty"`
	Priced            bool      `json:"priced"`
	ExactCost         bool      `json:"exactCost"`
	CreatedAt         time.Time `json:"createdAt"`
}

type TokenUsageModelSummary struct {
	Model       string  `json:"model"`
	TotalTokens int64   `json:"totalTokens"`
	CostUSD     float64 `json:"costUSD,omitempty"`
	MinCostUSD  float64 `json:"minCostUSD,omitempty"`
	MaxCostUSD  float64 `json:"maxCostUSD,omitempty"`
	Priced      bool    `json:"priced"`
	ExactCost   bool    `json:"exactCost"`
	Runs        int     `json:"runs"`
}

type TokenUsageSummary struct {
	TaskID        string                   `json:"taskId,omitempty"`
	TotalTokens   int64                    `json:"totalTokens"`
	CostUSD       float64                  `json:"costUSD,omitempty"`
	MinCostUSD    float64                  `json:"minCostUSD,omitempty"`
	MaxCostUSD    float64                  `json:"maxCostUSD,omitempty"`
	Priced        bool                     `json:"priced"`
	ExactCost     bool                     `json:"exactCost"`
	Models        []TokenUsageModelSummary `json:"models"`
	Records       []TokenUsageRecord       `json:"records,omitempty"`
	PricingNote   string                   `json:"pricingNote"`
	LastUpdatedAt *time.Time               `json:"lastUpdatedAt,omitempty"`
}

type MessageRole string

const (
	RoleUser     MessageRole = "user"
	RoleGardener MessageRole = "gardener"
	RoleSystem   MessageRole = "system"
)

type Message struct {
	ID        string      `json:"id"`
	Role      MessageRole `json:"role"`
	Content   string      `json:"content"`
	CreatedAt time.Time   `json:"createdAt"`
}

type TaskRuntime struct {
	Phase            string     `json:"phase"`
	Cue              string     `json:"cue"`
	Severity         string     `json:"severity"`
	IdleSeconds      int64      `json:"idleSeconds"`
	DurationSeconds  int64      `json:"durationSeconds"`
	RunningTrees     int        `json:"runningTrees"`
	FinishedTrees    int        `json:"finishedTrees"`
	TotalTrees       int        `json:"totalTrees"`
	LatestActivityAt *time.Time `json:"latestActivityAt,omitempty"`
	CanAskProgress   bool       `json:"canAskProgress"`
	CanResume        bool       `json:"canResume"`
}

type Task struct {
	ID                 string       `json:"id"`
	Title              string       `json:"title"`
	Prompt             string       `json:"prompt"`
	WorkspacePath      string       `json:"workspacePath"`
	ScratchPath        string       `json:"scratchPath,omitempty"`
	CLIEngine          CLIEngine    `json:"cliEngine"`
	ModelMode          ModelMode    `json:"modelMode"`
	Status             Status       `json:"status"`
	GardenerStatus     Status       `json:"gardenerStatus"`
	Forest             int          `json:"forest"`
	MaxTreesPerForest  int          `json:"maxTreesPerForest"`
	MaxConcurrentTrees int          `json:"maxConcurrentTrees"`
	StopRequested      bool         `json:"stopRequested"`
	AwaitingUserInput  bool         `json:"awaitingUserInput,omitempty"`
	SchedulePath       string       `json:"schedulePath"`
	LogPath            string       `json:"logPath"`
	Trees              []*Tree      `json:"trees"`
	Messages           []Message    `json:"messages"`
	GardenerProgress   []string     `json:"gardenerProgress"`
	LastProgressAt     *time.Time   `json:"lastProgressAt,omitempty"`
	LastWatchdogAt     *time.Time   `json:"lastWatchdogAt,omitempty"`
	Runtime            *TaskRuntime `json:"runtime,omitempty"`
	CreatedAt          time.Time    `json:"createdAt"`
	UpdatedAt          time.Time    `json:"updatedAt"`
}

type Tree struct {
	ID           string     `json:"id"`
	TaskID       string     `json:"taskId"`
	Forest       int        `json:"forest"`
	Name         string     `json:"name"`
	Objective    string     `json:"objective"`
	Prompt       string     `json:"prompt"`
	Scope        []string   `json:"scope"`
	IsValidation bool       `json:"isValidation"`
	Status       Status     `json:"status"`
	Progress     []string   `json:"progress"`
	FruitPath    string     `json:"fruitPath"`
	GoalPath     string     `json:"goalPath,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type CreateTaskRequest struct {
	Prompt        string `json:"prompt"`
	WorkspacePath string `json:"workspacePath"`
}

type SendMessageRequest struct {
	Content string `json:"content"`
}

type RenameTaskRequest struct {
	Title string `json:"title"`
}

type CreateTaskResponse struct {
	Task *Task `json:"task"`
}

type SettingsResponse struct {
	Settings AppSettings `json:"settings"`
}

type TreePlan struct {
	Name      string   `json:"name"`
	Objective string   `json:"objective"`
	Prompt    string   `json:"prompt"`
	Scope     []string `json:"scope"`
}

type GardenerPlan struct {
	MessageToUser         string     `json:"message_to_user"`
	ForestFinished        bool       `json:"forest_finished"`
	NeedsClarification    bool       `json:"needs_clarification,omitempty"`
	ClarificationQuestion string     `json:"clarification_question,omitempty"`
	Trees                 []TreePlan `json:"trees"`
}
