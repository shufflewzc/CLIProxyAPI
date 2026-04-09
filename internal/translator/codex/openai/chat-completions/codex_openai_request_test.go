package chat_completions

import (
	"encoding/json"
	"reflect"
	"testing"
)

// normalize parses JSON bytes into interface{} for deep-equal comparison,
// ignoring key order differences between the two implementations.
func normalize(t *testing.T, data []byte) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nraw: %s", err, data)
	}
	return v
}

// assertEquivalent compares new vs legacy output semantically.
func assertEquivalent(t *testing.T, name string, newOut, legacyOut []byte) {
	t.Helper()
	nv := normalize(t, newOut)
	lv := normalize(t, legacyOut)
	if !reflect.DeepEqual(nv, lv) {
		t.Errorf("case %q: outputs differ\n  new:    %s\n  legacy: %s", name, newOut, legacyOut)
	}
}

func TestConvertOpenAIRequestToCodex_UserStringContent(t *testing.T) {
	input := []byte(`{
		"model": "codex-mini-latest",
		"messages": [
			{"role": "user", "content": "Hello, world!"}
		]
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "user-string-content", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_SystemAndUser(t *testing.T) {
	input := []byte(`{
		"messages": [
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": "What is 2+2?"}
		]
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, true)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, true)
	assertEquivalent(t, "system-and-user", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_ContentArray(t *testing.T) {
	input := []byte(`{
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "What is in this image?"},
					{"type": "image_url", "image_url": {"url": "https://example.com/img.png"}}
				]
			}
		]
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "content-array-text-image", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_ContentArrayFile(t *testing.T) {
	input := []byte(`{
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "Read this file"},
					{"type": "file", "file": {"filename": "hello.txt", "file_data": "ZmlsZQ=="}}
				]
			}
		]
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "content-array-text-file", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_AssistantToolCalls(t *testing.T) {
	input := []byte(`{
		"messages": [
			{"role": "user", "content": "What's the weather?"},
			{
				"role": "assistant",
				"content": null,
				"tool_calls": [
					{
						"id": "call_abc123",
						"type": "function",
						"function": {"name": "get_weather", "arguments": "{\"location\": \"NYC\"}"}
					}
				]
			},
			{
				"role": "tool",
				"tool_call_id": "call_abc123",
				"content": "Sunny, 72F"
			}
		],
		"tools": [
			{
				"type": "function",
				"function": {
					"name": "get_weather",
					"description": "Get the current weather",
					"parameters": {
						"type": "object",
						"properties": {
							"location": {"type": "string"}
						},
						"required": ["location"]
					}
				}
			}
		]
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "assistant-tool-calls", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_ToolChoiceFunction(t *testing.T) {
	input := []byte(`{
		"messages": [
			{"role": "user", "content": "Search for me"}
		],
		"tools": [
			{
				"type": "function",
				"function": {
					"name": "search",
					"description": "Search the web",
					"parameters": {"type": "object", "properties": {}}
				}
			}
		],
		"tool_choice": {"type": "function", "function": {"name": "search"}}
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "tool-choice-function", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_ToolChoiceString(t *testing.T) {
	input := []byte(`{
		"messages": [{"role": "user", "content": "Hi"}],
		"tool_choice": "auto"
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "tool-choice-string", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_ResponseFormatJSONSchema(t *testing.T) {
	input := []byte(`{
		"messages": [{"role": "user", "content": "Give me JSON"}],
		"response_format": {
			"type": "json_schema",
			"json_schema": {
				"name": "my_schema",
				"strict": true,
				"schema": {
					"type": "object",
					"properties": {
						"result": {"type": "string"}
					},
					"required": ["result"]
				}
			}
		}
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "response-format-json-schema", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_ReasoningEffort(t *testing.T) {
	input := []byte(`{
		"messages": [{"role": "user", "content": "Think hard"}],
		"reasoning_effort": "high"
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "reasoning-effort-high", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_BuiltinTool(t *testing.T) {
	input := []byte(`{
		"messages": [{"role": "user", "content": "Search the web"}],
		"tools": [
			{"type": "web_search"}
		],
		"tool_choice": {"type": "web_search"}
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "builtin-tool", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_LongToolName(t *testing.T) {
	longName := "mcp__very_long_server_name__this_is_a_really_really_really_long_tool_name_that_exceeds_limit"
	input, _ := json.Marshal(map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "use the tool"},
		},
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        longName,
					"description": "A very long tool name",
					"parameters":  map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
		},
	})
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "long-tool-name", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_TextVerbosity(t *testing.T) {
	input := []byte(`{
		"messages": [{"role": "user", "content": "Be verbose"}],
		"text": {"verbosity": "verbose"}
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "text-verbosity", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_AssistantTextContent(t *testing.T) {
	input := []byte(`{
		"messages": [
			{"role": "user", "content": "Hello"},
			{"role": "assistant", "content": "Hi there! How can I help?"},
			{"role": "user", "content": "Tell me more"}
		]
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "assistant-text-content", newOut, legacyOut)
}

func TestConvertOpenAIRequestToCodex_FunctionToolWithStrict(t *testing.T) {
	input := []byte(`{
		"messages": [{"role": "user", "content": "Calculate"}],
		"tools": [
			{
				"type": "function",
				"function": {
					"name": "calculate",
					"description": "Do math",
					"strict": true,
					"parameters": {
						"type": "object",
						"properties": {
							"expression": {"type": "string"}
						},
						"required": ["expression"],
						"additionalProperties": false
					}
				}
			}
		]
	}`)
	newOut := ConvertOpenAIRequestToCodex("codex-mini-latest", input, false)
	legacyOut := ConvertOpenAIRequestToCodexLegacy("codex-mini-latest", input, false)
	assertEquivalent(t, "function-tool-with-strict", newOut, legacyOut)
}
