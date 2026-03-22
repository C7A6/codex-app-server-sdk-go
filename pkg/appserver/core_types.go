package appserver

import "encoding/json"

type ThreadStatus struct {
	Type        string   `json:"type"`
	ActiveFlags []string `json:"activeFlags,omitempty"`
}

type Thread struct {
	ID     string        `json:"id"`
	Name   *string       `json:"name,omitempty"`
	Status *ThreadStatus `json:"status,omitempty"`
}

type TurnError struct {
	Message           string          `json:"message"`
	CodexErrorInfo    json.RawMessage `json:"codexErrorInfo,omitempty"`
	AdditionalDetails json.RawMessage `json:"additionalDetails,omitempty"`
}

type Turn struct {
	ID     string       `json:"id"`
	Status string       `json:"status,omitempty"`
	Items  []ThreadItem `json:"items,omitempty"`
	Error  *TurnError   `json:"error,omitempty"`
}

type ThreadItem struct {
	ID               string          `json:"id"`
	Type             string          `json:"type"`
	Status           string          `json:"status,omitempty"`
	Text             string          `json:"text,omitempty"`
	Content          json.RawMessage `json:"content,omitempty"`
	Command          string          `json:"command,omitempty"`
	Cwd              string          `json:"cwd,omitempty"`
	AggregatedOutput string          `json:"aggregatedOutput,omitempty"`
	ExitCode         *int            `json:"exitCode,omitempty"`
	DurationMs       *int64          `json:"durationMs,omitempty"`
	Result           json.RawMessage `json:"result,omitempty"`
	Error            json.RawMessage `json:"error,omitempty"`
	Changes          json.RawMessage `json:"changes,omitempty"`
}

type ReviewStartResult struct {
	ReviewThreadID string `json:"reviewThreadId"`
	Turn           Turn   `json:"turn"`
}

type CommandExecResult struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

type ConfigValueMap map[string]any

type ConfigLayer map[string]any

type ConfigLayerMetadata map[string]any

type ConfigReadResult struct {
	Config  ConfigValueMap                 `json:"config"`
	Layers  []ConfigLayer                  `json:"layers,omitempty"`
	Origins map[string]ConfigLayerMetadata `json:"origins"`
}

type ModelListParams struct {
	Cursor        *string `json:"cursor,omitempty"`
	IncludeHidden *bool   `json:"includeHidden,omitempty"`
	Limit         *uint32 `json:"limit,omitempty"`
}

type ReasoningEffortOption struct {
	ReasoningEffort string `json:"reasoningEffort"`
	Description     string `json:"description"`
}

type ModelAvailabilityNux struct {
	Message string `json:"message"`
}

type ModelUpgradeInfo struct {
	Model             string  `json:"model"`
	MigrationMarkdown *string `json:"migrationMarkdown,omitempty"`
	ModelLink         *string `json:"modelLink,omitempty"`
	UpgradeCopy       *string `json:"upgradeCopy,omitempty"`
}

type ModelInfo struct {
	ID                        string                  `json:"id"`
	Model                     string                  `json:"model"`
	DisplayName               string                  `json:"displayName"`
	Description               string                  `json:"description"`
	Hidden                    bool                    `json:"hidden"`
	IsDefault                 bool                    `json:"isDefault"`
	DefaultReasoningEffort    string                  `json:"defaultReasoningEffort"`
	SupportedReasoningEfforts []ReasoningEffortOption `json:"supportedReasoningEfforts"`
	InputModalities           []string                `json:"inputModalities,omitempty"`
	SupportsPersonality       bool                    `json:"supportsPersonality,omitempty"`
	Upgrade                   *string                 `json:"upgrade,omitempty"`
	UpgradeInfo               *ModelUpgradeInfo       `json:"upgradeInfo,omitempty"`
	AvailabilityNux           *ModelAvailabilityNux   `json:"availabilityNux,omitempty"`
}

type ModelListResult struct {
	Data       []ModelInfo `json:"data"`
	NextCursor *string     `json:"nextCursor,omitempty"`
}
