// Copyright (C) 2026 Joey Kot <joey.kot.x@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed WITHOUT ANY WARRANTY; without even the
// implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
// See <https://www.gnu.org/licenses/> for more details.

package responses

import (
	"testing"

	"deepseek-responses-compatible/internal/state"
)

func TestNamespaceToolsFlattenAndRestore(t *testing.T) {
	adapter := Adapter{DefaultModel: "deepseek-v4-pro", Store: state.New()}
	payload := map[string]any{
		"input": "search",
		"tools": []any{
			map[string]any{
				"type":        "namespace",
				"name":        "mcp__firecrawl",
				"description": "Tools in namespace.",
				"tools": []any{
					map[string]any{
						"type":        "function",
						"name":        "firecrawl_web_search",
						"description": "Web Search Interface",
						"parameters":  map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}}},
					},
				},
			},
		},
		"tool_choice": map[string]any{"type": "function", "namespace": "mcp__firecrawl", "name": "firecrawl_web_search"},
	}

	prepared, err := adapter.Prepare(payload)
	if err != nil {
		t.Fatal(err)
	}
	chatPayload, toolNameMap := adapter.BuildDeepSeekPayload(payload, prepared.Messages)
	tools := chatPayload["tools"].([]map[string]any)
	deepseekName := "mcp__firecrawl__firecrawl_web_search"
	if got := tools[0]["function"].(map[string]any)["name"]; got != deepseekName {
		t.Fatalf("deepseek tool name = %v", got)
	}
	choice := chatPayload["tool_choice"].(map[string]any)["function"].(map[string]any)
	if got := choice["name"]; got != deepseekName {
		t.Fatalf("tool_choice = %v", got)
	}

	completion := map[string]any{"choices": []any{map[string]any{
		"finish_reason": "tool_calls",
		"message": map[string]any{
			"role":    "assistant",
			"content": "",
			"tool_calls": []any{map[string]any{
				"id":       "call_search",
				"type":     "function",
				"function": map[string]any{"name": deepseekName, "arguments": "{\"query\":\"DeepSeek\"}"},
			}},
		},
	}}}
	output, _, _, _ := adapter.OutputItemsFromChatCompletion(completion, toolNameMap)
	if got := output[0]["type"]; got != "function_call" {
		t.Fatalf("output type = %v", got)
	}
	if got := output[0]["name"]; got != "firecrawl_web_search" {
		t.Fatalf("restored name = %v", got)
	}
	if got := output[0]["namespace"]; got != "mcp__firecrawl" {
		t.Fatalf("restored namespace = %v", got)
	}
}

func TestPrepareDoesNotRegisterTransientInputItems(t *testing.T) {
	store := state.New()
	adapter := Adapter{DefaultModel: "deepseek-v4-pro", Store: store}

	_, err := adapter.Prepare(map[string]any{
		"input": []any{map[string]any{"id": "msg_transient", "type": "message", "role": "user", "content": "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if item, ok := store.Item("msg_transient"); ok {
		t.Fatalf("transient input item was registered: %#v", item)
	}
}

func TestReasoningIsPreservedForToolCallContext(t *testing.T) {
	adapter := Adapter{}
	completion := map[string]any{"choices": []any{map[string]any{
		"finish_reason": "tool_calls",
		"message": map[string]any{
			"role":              "assistant",
			"reasoning_content": "I need a tool.",
			"content":           "Let me check.",
			"tool_calls": []any{
				map[string]any{"id": "call_1", "type": "function", "function": map[string]any{"name": "fn", "arguments": "{}"}},
				map[string]any{"id": "call_2", "type": "function", "function": map[string]any{"name": "fn2", "arguments": "{\"x\":1}"}},
			},
		},
	}}}
	output, _, _, _ := adapter.OutputItemsFromChatCompletion(completion, nil)
	if got := output[0]["type"]; got != "reasoning" {
		t.Fatalf("reasoning item type = %v", got)
	}
	summary := output[0]["summary"].([]any)
	if got := summary[0].(map[string]any)["type"]; got != "summary_text" {
		t.Fatalf("summary type = %v", got)
	}
	if got := summary[0].(map[string]any)["text"]; got != "I need a tool." {
		t.Fatalf("summary text = %v", got)
	}
	messages := InputItemsToChatMessages(output)
	if len(messages) != 1 {
		t.Fatalf("messages len = %d", len(messages))
	}
	if got := messages[0]["reasoning_content"]; got != "I need a tool." {
		t.Fatalf("reasoning_content = %v", got)
	}
	calls := messages[0]["tool_calls"].([]any)
	if len(calls) != 2 {
		t.Fatalf("tool calls len = %d", len(calls))
	}
}
