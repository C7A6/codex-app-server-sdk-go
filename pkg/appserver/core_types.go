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
	Turns  []Turn        `json:"turns,omitempty"`
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

type ReviewDelivery string

const (
	ReviewDeliveryInline   ReviewDelivery = "inline"
	ReviewDeliveryDetached ReviewDelivery = "detached"
)

type ReviewTarget map[string]any

type ReviewStartParams struct {
	ThreadID string          `json:"threadId"`
	Target   ReviewTarget    `json:"target"`
	Delivery *ReviewDelivery `json:"delivery,omitempty"`
}

type CommandExecResult struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

type CommandExecTerminalSize struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

type CommandExecParams struct {
	Command            []string                 `json:"command"`
	Cwd                *string                  `json:"cwd,omitempty"`
	DisableOutputCap   bool                     `json:"disableOutputCap,omitempty"`
	DisableTimeout     bool                     `json:"disableTimeout,omitempty"`
	Env                map[string]*string       `json:"env,omitempty"`
	OutputBytesCap     *uint64                  `json:"outputBytesCap,omitempty"`
	ProcessID          *string                  `json:"processId,omitempty"`
	SandboxPolicy      any                      `json:"sandboxPolicy,omitempty"`
	Size               *CommandExecTerminalSize `json:"size,omitempty"`
	StreamStdin        bool                     `json:"streamStdin,omitempty"`
	StreamStdoutStderr bool                     `json:"streamStdoutStderr,omitempty"`
	TimeoutMs          *int64                   `json:"timeoutMs,omitempty"`
	TTY                bool                     `json:"tty,omitempty"`
}

type CommandExecWriteParams struct {
	ProcessID   string  `json:"processId"`
	CloseStdin  bool    `json:"closeStdin,omitempty"`
	DeltaBase64 *string `json:"deltaBase64,omitempty"`
}

type CommandExecWriteResult struct{}

type CommandExecResizeParams struct {
	ProcessID string                  `json:"processId"`
	Size      CommandExecTerminalSize `json:"size"`
}

type CommandExecResizeResult struct{}

type CommandExecTerminateParams struct {
	ProcessID string `json:"processId"`
}

type CommandExecTerminateResult struct{}

type ConfigValueMap map[string]any

type ConfigLayer map[string]any

type ConfigLayerMetadata map[string]any

type ConfigReadParams struct {
	Cwd           *string `json:"cwd,omitempty"`
	IncludeLayers bool    `json:"includeLayers,omitempty"`
}

type ConfigReadResult struct {
	Config  ConfigValueMap                 `json:"config"`
	Layers  []ConfigLayer                  `json:"layers,omitempty"`
	Origins map[string]ConfigLayerMetadata `json:"origins"`
}

type ConfigMergeStrategy string

const (
	ConfigMergeStrategyReplace ConfigMergeStrategy = "replace"
	ConfigMergeStrategyUpsert  ConfigMergeStrategy = "upsert"
)

type ConfigWriteParams struct {
	ExpectedVersion *string             `json:"expectedVersion,omitempty"`
	FilePath        *string             `json:"filePath,omitempty"`
	KeyPath         string              `json:"keyPath"`
	MergeStrategy   ConfigMergeStrategy `json:"mergeStrategy"`
	Value           any                 `json:"value"`
}

type ConfigEdit struct {
	KeyPath       string              `json:"keyPath"`
	MergeStrategy ConfigMergeStrategy `json:"mergeStrategy"`
	Value         any                 `json:"value"`
}

type ConfigBatchWriteParams struct {
	Edits            []ConfigEdit `json:"edits"`
	ExpectedVersion  *string      `json:"expectedVersion,omitempty"`
	FilePath         *string      `json:"filePath,omitempty"`
	ReloadUserConfig bool         `json:"reloadUserConfig,omitempty"`
}

type ConfigWriteStatus string

const (
	ConfigWriteStatusOK           ConfigWriteStatus = "ok"
	ConfigWriteStatusOKOverridden ConfigWriteStatus = "okOverridden"
)

type ConfigWriteOverriddenMetadata struct {
	EffectiveValue  any                 `json:"effectiveValue"`
	Message         string              `json:"message"`
	OverridingLayer ConfigLayerMetadata `json:"overridingLayer"`
}

type ConfigWriteResult struct {
	FilePath           string                         `json:"filePath"`
	OverriddenMetadata *ConfigWriteOverriddenMetadata `json:"overriddenMetadata,omitempty"`
	Status             ConfigWriteStatus              `json:"status"`
	Version            string                         `json:"version"`
}

type SkillsListExtraRootsForCwd struct {
	Cwd            string   `json:"cwd"`
	ExtraUserRoots []string `json:"extraUserRoots"`
}

type SkillsListParams struct {
	Cwds                 []string                     `json:"cwds,omitempty"`
	ForceReload          bool                         `json:"forceReload,omitempty"`
	PerCwdExtraUserRoots []SkillsListExtraRootsForCwd `json:"perCwdExtraUserRoots,omitempty"`
}

type SkillToolDependency struct {
	Command     *string `json:"command,omitempty"`
	Description *string `json:"description,omitempty"`
	Transport   *string `json:"transport,omitempty"`
	Type        string  `json:"type"`
	URL         *string `json:"url,omitempty"`
	Value       string  `json:"value"`
}

type SkillDependencies struct {
	Tools []SkillToolDependency `json:"tools"`
}

type SkillInterface struct {
	BrandColor       *string `json:"brandColor,omitempty"`
	DefaultPrompt    *string `json:"defaultPrompt,omitempty"`
	DisplayName      *string `json:"displayName,omitempty"`
	IconLarge        *string `json:"iconLarge,omitempty"`
	IconSmall        *string `json:"iconSmall,omitempty"`
	ShortDescription *string `json:"shortDescription,omitempty"`
}

type SkillMetadata struct {
	Dependencies     *SkillDependencies `json:"dependencies,omitempty"`
	Description      string             `json:"description"`
	Enabled          bool               `json:"enabled"`
	Interface        *SkillInterface    `json:"interface,omitempty"`
	Name             string             `json:"name"`
	Path             string             `json:"path"`
	Scope            string             `json:"scope"`
	ShortDescription *string            `json:"shortDescription,omitempty"`
}

type SkillErrorInfo struct {
	Message string `json:"message"`
	Path    string `json:"path"`
}

type SkillsListEntry struct {
	Cwd    string           `json:"cwd"`
	Errors []SkillErrorInfo `json:"errors"`
	Skills []SkillMetadata  `json:"skills"`
}

type SkillsListResult struct {
	Data []SkillsListEntry `json:"data"`
}

type SkillsConfigWriteParams struct {
	Path    string `json:"path"`
	Enabled bool   `json:"enabled"`
}

type SkillsConfigWriteResult struct {
	EffectiveEnabled bool `json:"effectiveEnabled"`
}

type PluginListParams struct {
	Cwds            []string `json:"cwds,omitempty"`
	ForceRemoteSync bool     `json:"forceRemoteSync,omitempty"`
}

type MarketplaceInterface struct {
	DisplayName *string `json:"displayName,omitempty"`
}

type PluginInterface struct {
	BrandColor        *string  `json:"brandColor,omitempty"`
	Capabilities      []string `json:"capabilities"`
	Category          *string  `json:"category,omitempty"`
	ComposerIcon      *string  `json:"composerIcon,omitempty"`
	DefaultPrompt     []string `json:"defaultPrompt,omitempty"`
	DeveloperName     *string  `json:"developerName,omitempty"`
	DisplayName       *string  `json:"displayName,omitempty"`
	Logo              *string  `json:"logo,omitempty"`
	LongDescription   *string  `json:"longDescription,omitempty"`
	PrivacyPolicyURL  *string  `json:"privacyPolicyUrl,omitempty"`
	Screenshots       []string `json:"screenshots"`
	ShortDescription  *string  `json:"shortDescription,omitempty"`
	TermsOfServiceURL *string  `json:"termsOfServiceUrl,omitempty"`
	WebsiteURL        *string  `json:"websiteUrl,omitempty"`
}

type PluginSource struct {
	Type string  `json:"type"`
	Path *string `json:"path,omitempty"`
}

type PluginSummary struct {
	AuthPolicy    string           `json:"authPolicy"`
	Enabled       bool             `json:"enabled"`
	ID            string           `json:"id"`
	InstallPolicy string           `json:"installPolicy"`
	Installed     bool             `json:"installed"`
	Interface     *PluginInterface `json:"interface,omitempty"`
	Name          string           `json:"name"`
	Source        PluginSource     `json:"source"`
}

type PluginMarketplaceEntry struct {
	Interface *MarketplaceInterface `json:"interface,omitempty"`
	Name      string                `json:"name"`
	Path      string                `json:"path"`
	Plugins   []PluginSummary       `json:"plugins"`
}

type PluginListResult struct {
	Marketplaces    []PluginMarketplaceEntry `json:"marketplaces"`
	RemoteSyncError *string                  `json:"remoteSyncError,omitempty"`
}

type PluginReadParams struct {
	MarketplacePath string `json:"marketplacePath"`
	PluginName      string `json:"pluginName"`
}

type AppSummary struct {
	Description *string `json:"description,omitempty"`
	ID          string  `json:"id"`
	InstallURL  *string `json:"installUrl,omitempty"`
	Name        string  `json:"name"`
}

type SkillSummary struct {
	Description      string          `json:"description"`
	Interface        *SkillInterface `json:"interface,omitempty"`
	Name             string          `json:"name"`
	Path             string          `json:"path"`
	ShortDescription *string         `json:"shortDescription,omitempty"`
}

type PluginDetail struct {
	Apps            []AppSummary   `json:"apps"`
	Description     *string        `json:"description,omitempty"`
	MarketplaceName string         `json:"marketplaceName"`
	MarketplacePath string         `json:"marketplacePath"`
	MCPServers      []string       `json:"mcpServers"`
	Skills          []SkillSummary `json:"skills"`
	Summary         PluginSummary  `json:"summary"`
}

type PluginReadResult struct {
	Plugin PluginDetail `json:"plugin"`
}

type AppsListParams struct {
	Cursor       *string `json:"cursor,omitempty"`
	ForceRefetch *bool   `json:"forceRefetch,omitempty"`
	Limit        *uint32 `json:"limit,omitempty"`
	ThreadID     *string `json:"threadId,omitempty"`
}

type AppBranding struct {
	Category          *string `json:"category,omitempty"`
	Developer         *string `json:"developer,omitempty"`
	IsDiscoverableApp bool    `json:"isDiscoverableApp"`
	PrivacyPolicy     *string `json:"privacyPolicy,omitempty"`
	TermsOfService    *string `json:"termsOfService,omitempty"`
	Website           *string `json:"website,omitempty"`
}

type AppReview struct {
	Status string `json:"status"`
}

type AppScreenshot struct {
	FileID     *string `json:"fileId,omitempty"`
	URL        *string `json:"url,omitempty"`
	UserPrompt string  `json:"userPrompt"`
}

type AppMetadata struct {
	Categories                 []string        `json:"categories,omitempty"`
	Developer                  *string         `json:"developer,omitempty"`
	FirstPartyRequiresInstall  *bool           `json:"firstPartyRequiresInstall,omitempty"`
	FirstPartyType             *string         `json:"firstPartyType,omitempty"`
	Review                     *AppReview      `json:"review,omitempty"`
	Screenshots                []AppScreenshot `json:"screenshots,omitempty"`
	SEODescription             *string         `json:"seoDescription,omitempty"`
	ShowInComposerWhenUnlinked *bool           `json:"showInComposerWhenUnlinked,omitempty"`
	SubCategories              []string        `json:"subCategories,omitempty"`
	Version                    *string         `json:"version,omitempty"`
	VersionID                  *string         `json:"versionId,omitempty"`
	VersionNotes               *string         `json:"versionNotes,omitempty"`
}

type AppInfo struct {
	AppMetadata         *AppMetadata      `json:"appMetadata,omitempty"`
	Branding            *AppBranding      `json:"branding,omitempty"`
	Description         *string           `json:"description,omitempty"`
	DistributionChannel *string           `json:"distributionChannel,omitempty"`
	ID                  string            `json:"id"`
	InstallURL          *string           `json:"installUrl,omitempty"`
	IsAccessible        bool              `json:"isAccessible"`
	IsEnabled           bool              `json:"isEnabled"`
	Labels              map[string]string `json:"labels,omitempty"`
	LogoURL             *string           `json:"logoUrl,omitempty"`
	LogoURLDark         *string           `json:"logoUrlDark,omitempty"`
	Name                string            `json:"name"`
	PluginDisplayNames  []string          `json:"pluginDisplayNames,omitempty"`
}

type AppsListResult struct {
	Data       []AppInfo `json:"data"`
	NextCursor *string   `json:"nextCursor,omitempty"`
}

type MCPOAuthLoginParams struct {
	Name        string   `json:"name"`
	Scopes      []string `json:"scopes,omitempty"`
	TimeoutSecs *int64   `json:"timeoutSecs,omitempty"`
}

type MCPOAuthLoginResult struct {
	AuthorizationURL string `json:"authorizationUrl"`
}

type MCPServerRefreshResult struct{}

type MCPAuthStatus string

const (
	MCPAuthStatusUnsupported MCPAuthStatus = "unsupported"
	MCPAuthStatusNotLoggedIn MCPAuthStatus = "notLoggedIn"
	MCPAuthStatusBearerToken MCPAuthStatus = "bearerToken"
	MCPAuthStatusOAuth       MCPAuthStatus = "oAuth"
)

type MCPServerStatusListParams struct {
	Cursor *string `json:"cursor,omitempty"`
	Limit  *uint32 `json:"limit,omitempty"`
}

type MCPResource struct {
	Meta        json.RawMessage   `json:"_meta,omitempty"`
	Annotations json.RawMessage   `json:"annotations,omitempty"`
	Description *string           `json:"description,omitempty"`
	Icons       []json.RawMessage `json:"icons,omitempty"`
	MimeType    *string           `json:"mimeType,omitempty"`
	Name        string            `json:"name"`
	Size        *int64            `json:"size,omitempty"`
	Title       *string           `json:"title,omitempty"`
	URI         string            `json:"uri"`
}

type MCPResourceTemplate struct {
	Annotations json.RawMessage `json:"annotations,omitempty"`
	Description *string         `json:"description,omitempty"`
	MimeType    *string         `json:"mimeType,omitempty"`
	Name        string          `json:"name"`
	Title       *string         `json:"title,omitempty"`
	URITemplate string          `json:"uriTemplate"`
}

type MCPTool struct {
	Meta         json.RawMessage   `json:"_meta,omitempty"`
	Annotations  json.RawMessage   `json:"annotations,omitempty"`
	Description  *string           `json:"description,omitempty"`
	Icons        []json.RawMessage `json:"icons,omitempty"`
	InputSchema  json.RawMessage   `json:"inputSchema"`
	Name         string            `json:"name"`
	OutputSchema json.RawMessage   `json:"outputSchema,omitempty"`
	Title        *string           `json:"title,omitempty"`
}

type MCPServerStatus struct {
	AuthStatus        MCPAuthStatus         `json:"authStatus"`
	Name              string                `json:"name"`
	ResourceTemplates []MCPResourceTemplate `json:"resourceTemplates"`
	Resources         []MCPResource         `json:"resources"`
	Tools             map[string]MCPTool    `json:"tools"`
}

type MCPServerStatusListResult struct {
	Data       []MCPServerStatus `json:"data"`
	NextCursor *string           `json:"nextCursor,omitempty"`
}

type ConfigRequirementsResidency string

const (
	ConfigRequirementsResidencyUS ConfigRequirementsResidency = "us"
)

type ConfigRequirements struct {
	AllowedApprovalPolicies []json.RawMessage            `json:"allowedApprovalPolicies,omitempty"`
	AllowedSandboxModes     []string                     `json:"allowedSandboxModes,omitempty"`
	AllowedWebSearchModes   []string                     `json:"allowedWebSearchModes,omitempty"`
	EnforceResidency        *ConfigRequirementsResidency `json:"enforceResidency,omitempty"`
	FeatureRequirements     map[string]bool              `json:"featureRequirements,omitempty"`
}

type ConfigRequirementsReadResult struct {
	Requirements *ConfigRequirements `json:"requirements"`
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

type ThreadStartParams struct {
	ApprovalPolicy        any            `json:"approvalPolicy,omitempty"`
	ApprovalsReviewer     *string        `json:"approvalsReviewer,omitempty"`
	BaseInstructions      *string        `json:"baseInstructions,omitempty"`
	Config                map[string]any `json:"config,omitempty"`
	Cwd                   *string        `json:"cwd,omitempty"`
	DeveloperInstructions *string        `json:"developerInstructions,omitempty"`
	ServiceTier           *string        `json:"serviceTier,omitempty"`
	Ephemeral             *bool          `json:"ephemeral,omitempty"`
	ServiceName           *string        `json:"serviceName,omitempty"`
	Personality           *string        `json:"personality,omitempty"`
	Model                 *string        `json:"model,omitempty"`
	ModelProvider         *string        `json:"modelProvider,omitempty"`
	Sandbox               *string        `json:"sandbox,omitempty"`
}

type ThreadStartResult struct {
	ApprovalPolicy    json.RawMessage `json:"approvalPolicy"`
	ApprovalsReviewer string          `json:"approvalsReviewer"`
	Cwd               string          `json:"cwd"`
	Model             string          `json:"model"`
	ModelProvider     string          `json:"modelProvider"`
	ReasoningEffort   *string         `json:"reasoningEffort,omitempty"`
	Sandbox           json.RawMessage `json:"sandbox"`
	ServiceTier       *string         `json:"serviceTier,omitempty"`
	Thread            Thread          `json:"thread"`
}

type ThreadResumeParams struct {
	ThreadID              string         `json:"threadId"`
	ApprovalPolicy        any            `json:"approvalPolicy,omitempty"`
	ApprovalsReviewer     *string        `json:"approvalsReviewer,omitempty"`
	BaseInstructions      *string        `json:"baseInstructions,omitempty"`
	Config                map[string]any `json:"config,omitempty"`
	Cwd                   *string        `json:"cwd,omitempty"`
	DeveloperInstructions *string        `json:"developerInstructions,omitempty"`
	Sandbox               *string        `json:"sandbox,omitempty"`
	Model                 *string        `json:"model,omitempty"`
	ModelProvider         *string        `json:"modelProvider,omitempty"`
	ServiceTier           *string        `json:"serviceTier,omitempty"`
	Personality           *string        `json:"personality,omitempty"`
}

type ThreadResumeResult = ThreadStartResult

type ThreadForkParams struct {
	ThreadID              string         `json:"threadId"`
	ApprovalPolicy        any            `json:"approvalPolicy,omitempty"`
	ApprovalsReviewer     *string        `json:"approvalsReviewer,omitempty"`
	BaseInstructions      *string        `json:"baseInstructions,omitempty"`
	Config                map[string]any `json:"config,omitempty"`
	Cwd                   *string        `json:"cwd,omitempty"`
	DeveloperInstructions *string        `json:"developerInstructions,omitempty"`
	Ephemeral             bool           `json:"ephemeral"`
	Model                 *string        `json:"model,omitempty"`
	ModelProvider         *string        `json:"modelProvider,omitempty"`
	ServiceTier           *string        `json:"serviceTier,omitempty"`
	Sandbox               *string        `json:"sandbox,omitempty"`
}

type ThreadForkResult = ThreadStartResult

type ThreadReadParams struct {
	ThreadID     string `json:"threadId"`
	IncludeTurns bool   `json:"includeTurns,omitempty"`
}

type ThreadReadResult struct {
	Thread Thread `json:"thread"`
}

type ThreadSortKey string

const (
	ThreadSortKeyCreatedAt ThreadSortKey = "created_at"
	ThreadSortKeyUpdatedAt ThreadSortKey = "updated_at"
)

type ThreadSourceKind string

const (
	ThreadSourceKindCLI                 ThreadSourceKind = "cli"
	ThreadSourceKindVSCode              ThreadSourceKind = "vscode"
	ThreadSourceKindExec                ThreadSourceKind = "exec"
	ThreadSourceKindAppServer           ThreadSourceKind = "appServer"
	ThreadSourceKindCustom              ThreadSourceKind = "custom"
	ThreadSourceKindSubAgent            ThreadSourceKind = "subAgent"
	ThreadSourceKindSubAgentReview      ThreadSourceKind = "subAgentReview"
	ThreadSourceKindSubAgentCompact     ThreadSourceKind = "subAgentCompact"
	ThreadSourceKindSubAgentThreadSpawn ThreadSourceKind = "subAgentThreadSpawn"
	ThreadSourceKindSubAgentOther       ThreadSourceKind = "subAgentOther"
	ThreadSourceKindUnknown             ThreadSourceKind = "unknown"
)

type ThreadListParams struct {
	Archived       *bool              `json:"archived,omitempty"`
	Cursor         *string            `json:"cursor,omitempty"`
	Cwd            *string            `json:"cwd,omitempty"`
	Limit          *uint32            `json:"limit,omitempty"`
	ModelProviders []string           `json:"modelProviders,omitempty"`
	SearchTerm     *string            `json:"searchTerm,omitempty"`
	SortKey        *ThreadSortKey     `json:"sortKey,omitempty"`
	SourceKinds    []ThreadSourceKind `json:"sourceKinds,omitempty"`
}

type ThreadListResult struct {
	Data       []Thread `json:"data"`
	NextCursor *string  `json:"nextCursor,omitempty"`
}

type ThreadLoadedListParams struct {
	Cursor *string `json:"cursor,omitempty"`
	Limit  *uint32 `json:"limit,omitempty"`
}

type ThreadLoadedListResult struct {
	Data       []string `json:"data"`
	NextCursor *string  `json:"nextCursor,omitempty"`
}

type ThreadSetNameParams struct {
	ThreadID string `json:"threadId"`
	Name     string `json:"name"`
}

type ThreadSetNameResult struct{}

type ThreadArchiveParams struct {
	ThreadID string `json:"threadId"`
}

type ThreadArchiveResult struct{}

type ThreadUnarchiveParams struct {
	ThreadID string `json:"threadId"`
}

type ThreadUnarchiveResult struct {
	Thread Thread `json:"thread"`
}

type ThreadUnsubscribeParams struct {
	ThreadID string `json:"threadId"`
}

type ThreadUnsubscribeStatus string

const (
	ThreadUnsubscribeStatusNotLoaded     ThreadUnsubscribeStatus = "notLoaded"
	ThreadUnsubscribeStatusNotSubscribed ThreadUnsubscribeStatus = "notSubscribed"
	ThreadUnsubscribeStatusUnsubscribed  ThreadUnsubscribeStatus = "unsubscribed"
)

type ThreadUnsubscribeResult struct {
	Status ThreadUnsubscribeStatus `json:"status"`
}

type ThreadCompactStartParams struct {
	ThreadID string `json:"threadId"`
}

type ThreadCompactStartResult struct{}

type ThreadRollbackParams struct {
	ThreadID string `json:"threadId"`
	NumTurns uint32 `json:"numTurns"`
}

type ThreadRollbackResult struct {
	Thread Thread `json:"thread"`
}

type TurnStartInputItem map[string]any

type TurnStartParams struct {
	ThreadID          string               `json:"threadId"`
	Input             []TurnStartInputItem `json:"input"`
	ApprovalPolicy    any                  `json:"approvalPolicy,omitempty"`
	ApprovalsReviewer *string              `json:"approvalsReviewer,omitempty"`
	Cwd               *string              `json:"cwd,omitempty"`
	Effort            *string              `json:"effort,omitempty"`
	Model             *string              `json:"model,omitempty"`
	OutputSchema      any                  `json:"outputSchema,omitempty"`
	Personality       *string              `json:"personality,omitempty"`
	SandboxPolicy     any                  `json:"sandboxPolicy,omitempty"`
	ServiceTier       *string              `json:"serviceTier,omitempty"`
	Summary           *string              `json:"summary,omitempty"`
}

type TurnStartResult struct {
	Turn Turn `json:"turn"`
}

type TurnSteerParams struct {
	ThreadID       string               `json:"threadId"`
	ExpectedTurnID string               `json:"expectedTurnId"`
	Input          []TurnStartInputItem `json:"input"`
}

type TurnSteerResult struct {
	TurnID string `json:"turnId"`
}

type TurnInterruptParams struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type TurnInterruptResult struct{}
