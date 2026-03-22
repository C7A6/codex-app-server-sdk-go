package appserver

import (
	"encoding/json"
	"testing"
)

type goldenRequestEnvelope struct {
	Method string `json:"method"`
	ID     int    `json:"id"`
	Params any    `json:"params"`
}

type goldenResultEnvelope[T any] struct {
	ID     int `json:"id"`
	Result T   `json:"result"`
}

type goldenErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type goldenErrorEnvelope struct {
	ID    int                `json:"id"`
	Error goldenErrorPayload `json:"error"`
}

type goldenThreadStartEnvelope struct {
	Thread Thread `json:"thread"`
}

type goldenAccountReadEnvelope struct {
	Account            *Account `json:"account"`
	RequiresOpenAIAuth bool     `json:"requiresOpenaiAuth"`
}

type goldenServerRequestResolved struct {
	ThreadID  string `json:"threadId"`
	RequestID string `json:"requestId"`
}

type goldenNetworkApprovalContext struct {
	Host     string  `json:"host"`
	Port     *int    `json:"port,omitempty"`
	Protocol string  `json:"protocol"`
	Command  *string `json:"command,omitempty"`
}

type goldenCommandApprovalRequest struct {
	ItemID                 string                        `json:"itemId"`
	ThreadID               string                        `json:"threadId"`
	TurnID                 string                        `json:"turnId"`
	Reason                 *string                       `json:"reason,omitempty"`
	Command                []string                      `json:"command,omitempty"`
	Cwd                    *string                       `json:"cwd,omitempty"`
	AvailableDecisions     []string                      `json:"availableDecisions,omitempty"`
	NetworkApprovalContext *goldenNetworkApprovalContext `json:"networkApprovalContext,omitempty"`
}

type goldenFileChangeApprovalRequest struct {
	ItemID    string  `json:"itemId"`
	ThreadID  string  `json:"threadId"`
	TurnID    string  `json:"turnId"`
	Reason    *string `json:"reason,omitempty"`
	GrantRoot *string `json:"grantRoot,omitempty"`
}

type goldenAppListUpdatedEvent struct {
	Data []AppInfo `json:"data"`
}

func TestGoldenInitializeRequestEncoding(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(goldenRequestEnvelope{
		Method: "initialize",
		ID:     1,
		Params: InitializeParams{
			ClientInfo: ClientInfo{
				Name:    "my_client",
				Title:   "My Client",
				Version: "0.1.0",
			},
			Capabilities: &Capabilities{
				ExperimentalAPI:           true,
				OptOutNotificationMethods: []string{"thread/started", "item/agentMessage/delta"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	expected := `{"method":"initialize","id":1,"params":{"clientInfo":{"name":"my_client","title":"My Client","version":"0.1.0"},"capabilities":{"experimentalApi":true,"optOutNotificationMethods":["thread/started","item/agentMessage/delta"]}}}`
	if string(payload) != expected {
		t.Fatalf("unexpected initialize payload:\nwant: %s\ngot:  %s", expected, string(payload))
	}
}

func TestGoldenInitializeResultAndProtocolErrors(t *testing.T) {
	t.Parallel()

	var initialized goldenResultEnvelope[InitializeResult]
	if err := json.Unmarshal([]byte(`{"id":0,"result":{"userAgent":"Codex/0.116.0","platformFamily":"unix","platformOs":"linux"}}`), &initialized); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}
	if initialized.Result.UserAgent != "Codex/0.116.0" || initialized.Result.PlatformFamily != "unix" || initialized.Result.PlatformOS != "linux" {
		t.Fatalf("unexpected initialize result: %#v", initialized)
	}

	for name, raw := range map[string]string{
		"not initialized":     `{"id":1,"error":{"code":-32000,"message":"Not initialized"}}`,
		"already initialized": `{"id":1,"error":{"code":-32000,"message":"Already initialized"}}`,
	} {
		name := name
		raw := raw
		t.Run(name, func(t *testing.T) {
			var envelope goldenErrorEnvelope
			if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
				t.Fatalf("unmarshal error envelope: %v", err)
			}
			if envelope.Error.Message == "" || envelope.Error.Code == 0 {
				t.Fatalf("unexpected error envelope: %#v", envelope)
			}
		})
	}
}

func TestGoldenThreadAndTurnFixtures(t *testing.T) {
	t.Parallel()

	var threadStarted goldenResultEnvelope[goldenThreadStartEnvelope]
	if err := json.Unmarshal([]byte(`{"id":10,"result":{"thread":{"id":"thr_123","preview":"","ephemeral":false,"modelProvider":"openai","createdAt":1730910000}}}`), &threadStarted); err != nil {
		t.Fatalf("unmarshal thread/start response: %v", err)
	}
	if threadStarted.Result.Thread.ID != "thr_123" {
		t.Fatalf("unexpected thread/start result: %#v", threadStarted)
	}

	event, err := DecodeNotificationEvent(Notification{Method: MethodThreadStarted, Params: json.RawMessage(`{"thread":{"id":"thr_123"}}`)})
	if err != nil {
		t.Fatalf("DecodeNotificationEvent thread/started: %v", err)
	}
	threadEvent, ok := event.(*ThreadStartedEvent)
	if !ok || threadEvent.Thread.ID != "thr_123" {
		t.Fatalf("unexpected thread/started event: %#v", event)
	}

	var readThread goldenResultEnvelope[ThreadReadResult]
	if err := json.Unmarshal([]byte(`{"id":19,"result":{"thread":{"id":"thr_123","name":"Bug bash notes","ephemeral":false,"status":{"type":"notLoaded"},"turns":[]}}}`), &readThread); err != nil {
		t.Fatalf("unmarshal thread/read response: %v", err)
	}
	if readThread.Result.Thread.Status == nil || readThread.Result.Thread.Status.Type != "notLoaded" {
		t.Fatalf("unexpected thread/read result: %#v", readThread)
	}

	completedEvent, err := DecodeNotificationEvent(Notification{Method: MethodTurnCompleted, Params: json.RawMessage(`{
		"threadId":"thr_123",
		"turn":{
			"id":"turn_456",
			"status":"failed",
			"items":[],
			"error":{
				"message":"upstream error",
				"codexErrorInfo":{"type":"HttpConnectionFailed","httpStatusCode":502}
			}
		}
	}`)})
	if err != nil {
		t.Fatalf("DecodeNotificationEvent turn/completed: %v", err)
	}
	completed, ok := completedEvent.(*TurnCompletedEvent)
	if !ok || completed.Turn.Status != "failed" || completed.Turn.Error == nil {
		t.Fatalf("unexpected turn/completed event: %#v", completedEvent)
	}

	var codexError map[string]any
	if err := json.Unmarshal(completed.Turn.Error.CodexErrorInfo, &codexError); err != nil {
		t.Fatalf("unmarshal codex error info: %v", err)
	}
	if codexError["type"] != "HttpConnectionFailed" {
		t.Fatalf("unexpected codex error info: %#v", codexError)
	}
}

func TestGoldenTurnAndItemDeltaFixtures(t *testing.T) {
	t.Parallel()

	planEvent, err := DecodeNotificationEvent(Notification{Method: MethodTurnPlanUpdated, Params: json.RawMessage(`{
		"turnId":"turn_456",
		"explanation":"work plan",
		"plan":[
			{"step":"Inspect repo","status":"completed"},
			{"step":"Write tests","status":"inProgress"}
		],
		"items":[]
	}`)})
	if err != nil {
		t.Fatalf("DecodeNotificationEvent turn/plan/updated: %v", err)
	}
	planUpdated, ok := planEvent.(*TurnPlanUpdatedEvent)
	if !ok || planUpdated.TurnID != "turn_456" {
		t.Fatalf("unexpected turn/plan/updated event: %#v", planEvent)
	}

	diffEvent, err := DecodeNotificationEvent(Notification{Method: MethodTurnDiffUpdated, Params: json.RawMessage(`{
		"threadId":"thr_123",
		"turnId":"turn_456",
		"diff":"--- a/file.txt\n+++ b/file.txt\n@@ -1 +1 @@\n-old\n+new\n"
	}`)})
	if err != nil {
		t.Fatalf("DecodeNotificationEvent turn/diff/updated: %v", err)
	}
	diffUpdated, ok := diffEvent.(*TurnDiffUpdatedEvent)
	if !ok || diffUpdated.ThreadID != "thr_123" || diffUpdated.TurnID != "turn_456" {
		t.Fatalf("unexpected turn/diff/updated event: %#v", diffEvent)
	}

	reasoningPart, err := DecodeNotificationEvent(Notification{Method: MethodItemReasoningPartAdded, Params: json.RawMessage(`{"itemId":"item_1","summaryIndex":1,"threadId":"thr_123","turnId":"turn_456"}`)})
	if err != nil {
		t.Fatalf("DecodeNotificationEvent item/reasoning/summaryPartAdded: %v", err)
	}
	if typed, ok := reasoningPart.(*ReasoningSummaryPartAddedEvent); !ok || typed.SummaryIndex != 1 {
		t.Fatalf("unexpected reasoning summary part event: %#v", reasoningPart)
	}

	reasoningSummary, err := DecodeNotificationEvent(Notification{Method: MethodItemReasoningSummaryDelta, Params: json.RawMessage(`{"itemId":"item_1","summaryIndex":1,"delta":"considering options","threadId":"thr_123","turnId":"turn_456"}`)})
	if err != nil {
		t.Fatalf("DecodeNotificationEvent item/reasoning/summaryTextDelta: %v", err)
	}
	if typed, ok := reasoningSummary.(*ReasoningSummaryTextDeltaEvent); !ok || typed.Delta != "considering options" {
		t.Fatalf("unexpected reasoning summary delta event: %#v", reasoningSummary)
	}

	reasoningText, err := DecodeNotificationEvent(Notification{Method: MethodItemReasoningTextDelta, Params: json.RawMessage(`{"itemId":"item_1","contentIndex":0,"delta":"raw chain","threadId":"thr_123","turnId":"turn_456"}`)})
	if err != nil {
		t.Fatalf("DecodeNotificationEvent item/reasoning/textDelta: %v", err)
	}
	if typed, ok := reasoningText.(*ReasoningTextDeltaEvent); !ok || typed.Delta != "raw chain" {
		t.Fatalf("unexpected reasoning text delta event: %#v", reasoningText)
	}

	commandDelta, err := DecodeNotificationEvent(Notification{Method: MethodItemCommandOutputDelta, Params: json.RawMessage(`{"itemId":"cmd_1","delta":"hello\n","stream":"stdout","threadId":"thr_123","turnId":"turn_456"}`)})
	if err != nil {
		t.Fatalf("DecodeNotificationEvent item/commandExecution/outputDelta: %v", err)
	}
	if typed, ok := commandDelta.(*CommandExecutionOutputDeltaEvent); !ok || typed.Stream != "stdout" || typed.Delta != "hello\n" {
		t.Fatalf("unexpected command output delta event: %#v", commandDelta)
	}

	fileDelta, err := DecodeNotificationEvent(Notification{Method: MethodItemFileChangeOutputDelta, Params: json.RawMessage(`{"itemId":"file_1","delta":"*** Begin Patch\n*** End Patch\n","threadId":"thr_123","turnId":"turn_456"}`)})
	if err != nil {
		t.Fatalf("DecodeNotificationEvent item/fileChange/outputDelta: %v", err)
	}
	if typed, ok := fileDelta.(*FileChangeOutputDeltaEvent); !ok || typed.ItemID != "file_1" {
		t.Fatalf("unexpected file change output delta event: %#v", fileDelta)
	}
}

func TestGoldenApprovalAndServerRequestFixtures(t *testing.T) {
	t.Parallel()

	var commandApproval goldenCommandApprovalRequest
	if err := json.Unmarshal([]byte(`{
		"itemId":"cmd_1",
		"threadId":"thr_123",
		"turnId":"turn_456",
		"reason":"Network access required",
		"command":["curl","https://api.openai.com"],
		"cwd":"/tmp/work",
		"availableDecisions":["accept","decline","cancel"],
		"networkApprovalContext":{"host":"api.openai.com","protocol":"https","port":443}
	}`), &commandApproval); err != nil {
		t.Fatalf("unmarshal command approval request: %v", err)
	}
	if commandApproval.NetworkApprovalContext == nil || commandApproval.NetworkApprovalContext.Host != "api.openai.com" {
		t.Fatalf("unexpected command approval request: %#v", commandApproval)
	}

	var fileApproval goldenFileChangeApprovalRequest
	if err := json.Unmarshal([]byte(`{
		"itemId":"file_1",
		"threadId":"thr_123",
		"turnId":"turn_456",
		"reason":"Write files",
		"grantRoot":"/tmp/work"
	}`), &fileApproval); err != nil {
		t.Fatalf("unmarshal file approval request: %v", err)
	}
	if fileApproval.GrantRoot == nil || *fileApproval.GrantRoot != "/tmp/work" {
		t.Fatalf("unexpected file approval request: %#v", fileApproval)
	}

	var resolved goldenServerRequestResolved
	if err := json.Unmarshal([]byte(`{"threadId":"thr_123","requestId":"req_1"}`), &resolved); err != nil {
		t.Fatalf("unmarshal server request resolved: %v", err)
	}
	if resolved.ThreadID != "thr_123" || resolved.RequestID != "req_1" {
		t.Fatalf("unexpected server request resolved payload: %#v", resolved)
	}

	var requestUserInput ToolRequestUserInputParams
	if err := json.Unmarshal([]byte(`{
		"itemId":"item-1",
		"threadId":"thr-1",
		"turnId":"turn-1",
		"questions":[
			{
				"header":"Pick",
				"id":"choice",
				"question":"Choose one",
				"options":[
					{"label":"A","description":"Option A"},
					{"label":"B","description":"Option B","isOther":true}
				]
			}
		]
	}`), &requestUserInput); err != nil {
		t.Fatalf("unmarshal tool request user input params: %v", err)
	}
	if len(requestUserInput.Questions) != 1 || len(requestUserInput.Questions[0].Options) != 2 {
		t.Fatalf("unexpected tool request user input payload: %#v", requestUserInput)
	}
}

func TestGoldenAuthAndAsyncFixtures(t *testing.T) {
	t.Parallel()

	for name, raw := range map[string]string{
		"unauthenticated": `{"id":1,"result":{"account":null,"requiresOpenaiAuth":false}}`,
		"api key":         `{"id":1,"result":{"account":{"type":"apiKey"},"requiresOpenaiAuth":true}}`,
		"chatgpt":         `{"id":1,"result":{"account":{"type":"chatgpt","email":"user@example.com","planType":"pro"},"requiresOpenaiAuth":true}}`,
	} {
		name := name
		raw := raw
		t.Run(name, func(t *testing.T) {
			var envelope goldenResultEnvelope[goldenAccountReadEnvelope]
			if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
				t.Fatalf("unmarshal account/read fixture: %v", err)
			}
			if envelope.Result.Account != nil && envelope.Result.Account.Type == "" {
				t.Fatalf("unexpected account/read fixture: %#v", envelope)
			}
		})
	}

	var rateLimits goldenResultEnvelope[RateLimitsReadResult]
	if err := json.Unmarshal([]byte(`{
		"id":6,
		"result":{
			"rateLimits":{"limitId":"codex","limitName":null,"primary":{"usedPercent":25,"windowDurationMins":15,"resetsAt":1730947200},"secondary":null},
			"rateLimitsByLimitId":{
				"codex":{"limitId":"codex","limitName":null,"primary":{"usedPercent":25,"windowDurationMins":15,"resetsAt":1730947200},"secondary":null},
				"codex_other":{"limitId":"codex_other","limitName":"codex_other","primary":{"usedPercent":42,"windowDurationMins":60,"resetsAt":1730950800},"secondary":null}
			}
		}
	}`), &rateLimits); err != nil {
		t.Fatalf("unmarshal rate limits fixture: %v", err)
	}
	if rateLimits.Result.RateLimits == nil || len(rateLimits.Result.RateLimitsByLimitID) != 2 {
		t.Fatalf("unexpected rate limits fixture: %#v", rateLimits)
	}

	rateLimitsUpdated, err := DecodeNotificationEvent(Notification{Method: MethodAccountRateLimitsUpdated, Params: json.RawMessage(`{"rateLimits":{"limitId":"codex","primary":{"usedPercent":31,"windowDurationMins":15,"resetsAt":1730948100}}}`)})
	if err != nil {
		t.Fatalf("DecodeNotificationEvent account/rateLimits/updated: %v", err)
	}
	if _, ok := rateLimitsUpdated.(*AccountRateLimitsUpdatedEvent); !ok {
		t.Fatalf("unexpected rate limits updated event: %#v", rateLimitsUpdated)
	}

	var appUpdated goldenAppListUpdatedEvent
	if err := json.Unmarshal([]byte(`{
		"data":[{
			"id":"demo-app",
			"name":"Demo App",
			"description":"Example connector for documentation.",
			"logoUrl":"https://example.com/demo-app.png",
			"logoUrlDark":null,
			"distributionChannel":null,
			"branding":null,
			"appMetadata":null,
			"labels":null,
			"installUrl":"https://chatgpt.com/apps/demo-app/demo-app",
			"isAccessible":true,
			"isEnabled":true
		}]
	}`), &appUpdated); err != nil {
		t.Fatalf("unmarshal app/list/updated fixture: %v", err)
	}
	if len(appUpdated.Data) != 1 || appUpdated.Data[0].ID != "demo-app" {
		t.Fatalf("unexpected app/list/updated fixture: %#v", appUpdated)
	}

	mcpCompleted, err := DecodeNotificationEvent(Notification{Method: MethodMCPOAuthLoginCompleted, Params: json.RawMessage(`{"name":"demo-mcp","success":false,"error":"denied"}`)})
	if err != nil {
		t.Fatalf("DecodeNotificationEvent mcpServer/oauthLogin/completed: %v", err)
	}
	if typed, ok := mcpCompleted.(*MCPOAuthLoginCompletedEvent); !ok || typed.Name != "demo-mcp" || typed.Error == nil {
		t.Fatalf("unexpected mcp oauth completed event: %#v", mcpCompleted)
	}

	windowsCompleted, err := DecodeNotificationEvent(Notification{Method: MethodWindowsSandboxCompleted, Params: json.RawMessage(`{"mode":"elevated","success":true,"error":null}`)})
	if err != nil {
		t.Fatalf("DecodeNotificationEvent windowsSandbox/setupCompleted: %v", err)
	}
	if typed, ok := windowsCompleted.(*WindowsSandboxSetupCompletedEvent); !ok || typed.Mode != WindowsSandboxSetupModeElevated || !typed.Success {
		t.Fatalf("unexpected windows sandbox completed event: %#v", windowsCompleted)
	}
}
