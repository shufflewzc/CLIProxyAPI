package responses

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func ConvertOpenAIResponsesRequestToCodex(modelName string, inputRawJSON []byte, _ bool) []byte {
	_ = modelName

	var m map[string]json.RawMessage
	if err := json.Unmarshal(inputRawJSON, &m); err != nil {
		return inputRawJSON
	}

	if inputRaw, ok := m["input"]; ok {
		trimmed := bytes.TrimSpace(inputRaw)
		if len(trimmed) > 0 {
			switch trimmed[0] {
			case '"':
				var s string
				if err := json.Unmarshal(trimmed, &s); err == nil {
					wrapped, err := json.Marshal([]codexInputMessage{{
						Type: "message",
						Role: "user",
						Content: []codexInputContent{
							{Type: "input_text", Text: s},
						},
					}})
					if err == nil {
						m["input"] = wrapped
					}
				}
			case '[':
				if rewritten, changed := rewriteInputArraySystemRole(trimmed); changed {
					m["input"] = rewritten
				}
			}
		}
	}

	m["stream"] = json.RawMessage("true")
	m["store"] = json.RawMessage("false")
	m["parallel_tool_calls"] = json.RawMessage("true")
	m["include"] = json.RawMessage(`["reasoning.encrypted_content"]`)

	delete(m, "max_output_tokens")
	delete(m, "max_completion_tokens")
	delete(m, "temperature")
	delete(m, "top_p")
	delete(m, "truncation")
	delete(m, "user")
	delete(m, "context_management")

	out, err := json.Marshal(m)
	if err != nil {
		return inputRawJSON
	}
	return out
}

type codexInputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexInputMessage struct {
	Type    string              `json:"type"`
	Role    string              `json:"role"`
	Content []codexInputContent `json:"content"`
}

func rewriteInputArraySystemRole(inputRaw json.RawMessage) (json.RawMessage, bool) {
	var elems []json.RawMessage
	if err := json.Unmarshal(inputRaw, &elems); err != nil {
		return inputRaw, false
	}

	changed := false
	for i, elem := range elems {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(elem, &obj); err != nil {
			continue
		}
		roleRaw, ok := obj["role"]
		if !ok {
			continue
		}
		var role string
		if err := json.Unmarshal(roleRaw, &role); err != nil {
			continue
		}
		if role != "system" {
			continue
		}

		obj["role"] = json.RawMessage(`"developer"`)
		newElem, err := json.Marshal(obj)
		if err != nil {
			continue
		}
		elems[i] = newElem
		changed = true
	}

	if !changed {
		return inputRaw, false
	}
	out, err := json.Marshal(elems)
	if err != nil {
		return inputRaw, false
	}
	return out, true
}

// applyResponsesCompactionCompatibility handles OpenAI Responses context_management.compaction
// for Codex upstream compatibility.
//
// Codex /responses currently rejects context_management with:
// {"detail":"Unsupported parameter: context_management"}.
//
// Compatibility strategy:
// 1) Remove context_management before forwarding to Codex upstream.
func applyResponsesCompactionCompatibility(rawJSON []byte) []byte {
	if !gjson.GetBytes(rawJSON, "context_management").Exists() {
		return rawJSON
	}

	rawJSON, _ = sjson.DeleteBytes(rawJSON, "context_management")
	return rawJSON
}

// convertSystemRoleToDeveloper traverses the input array and converts any message items
// with role "system" to role "developer". This is necessary because Codex API does not
// accept "system" role in the input array.
func convertSystemRoleToDeveloper(rawJSON []byte) []byte {
	inputResult := gjson.GetBytes(rawJSON, "input")
	if !inputResult.IsArray() {
		return rawJSON
	}

	inputArray := inputResult.Array()
	needChange := false
	for i := 0; i < len(inputArray); i++ {
		role := inputArray[i].Get("role")
		if role.Type == gjson.String && role.Str == "system" {
			needChange = true
			break
		}
	}
	if !needChange {
		return rawJSON
	}

	result := rawJSON
	for i := 0; i < len(inputArray); i++ {
		role := inputArray[i].Get("role")
		if role.Type == gjson.String && role.Str == "system" {
			result, _ = sjson.SetBytes(result, fmt.Sprintf("input.%d.role", i), "developer")
		}
	}

	return result
}
