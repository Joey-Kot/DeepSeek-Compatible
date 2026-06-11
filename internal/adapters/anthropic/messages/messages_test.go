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

package messages

import "testing"

func TestBuildDeepSeekPayloadMapsAnthropicToolsAndThinking(t *testing.T) {
	adapter := Adapter{DefaultModel: "deepseek-v4-pro"}
	payload := map[string]any{
		"model":      "deepseek-v4-pro",
		"system":     "Be brief.",
		"max_tokens": 32,
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "Weather?"}}},
			map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": "toolu_1", "name": "get_weather", "input": map[string]any{"city": "Hangzhou"}}}},
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "tool_result", "tool_use_id": "toolu_1", "content": "24C"}}},
		},
		"tools":       []any{map[string]any{"name": "get_weather", "description": "Get weather.", "input_schema": map[string]any{"type": "object"}}},
		"tool_choice": map[string]any{"type": "tool", "name": "get_weather"},
		"thinking":    map[string]any{"type": "enabled", "budget_tokens": 1024},
	}

	prepared, err := adapter.BuildDeepSeekPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	messages := prepared.ChatPayload["messages"].([]map[string]any)
	if messages[0]["role"] != "system" || messages[0]["content"] != "Be brief." {
		t.Fatalf("system message = %#v", messages[0])
	}
	if messages[2]["role"] != "assistant" {
		t.Fatalf("assistant tool message = %#v", messages[2])
	}
	if messages[3]["role"] != "tool" || messages[3]["tool_call_id"] != "toolu_1" {
		t.Fatalf("tool result = %#v", messages[3])
	}
	tools := prepared.ChatPayload["tools"].([]map[string]any)
	if tools[0]["function"].(map[string]any)["name"] != "get_weather" {
		t.Fatalf("tools = %#v", tools)
	}
	choice := prepared.ChatPayload["tool_choice"].(map[string]any)["function"].(map[string]any)
	if choice["name"] != "get_weather" {
		t.Fatalf("tool_choice = %#v", choice)
	}
	if prepared.ChatPayload["thinking"].(map[string]any)["type"] != "enabled" {
		t.Fatalf("thinking = %#v", prepared.ChatPayload["thinking"])
	}
}

func TestResponseFromDeepSeekMapsAnthropicContent(t *testing.T) {
	completion := map[string]any{
		"id":    "chat_1",
		"model": "deepseek-v4-pro",
		"choices": []any{map[string]any{
			"finish_reason": "tool_calls",
			"message": map[string]any{
				"role":              "assistant",
				"reasoning_content": "Need tool.",
				"content":           "Checking.",
				"tool_calls":        []any{map[string]any{"id": "call_1", "type": "function", "function": map[string]any{"name": "get_weather", "arguments": "{\"city\":\"Hangzhou\"}"}}},
			},
		}},
		"usage": map[string]any{"prompt_tokens": 5, "completion_tokens": 3},
	}
	response := ResponseFromDeepSeek(completion, nil, "deepseek-v4-pro")
	if response["type"] != "message" || response["stop_reason"] != "tool_use" {
		t.Fatalf("response = %#v", response)
	}
	blocks := response["content"].([]any)
	if blocks[0].(map[string]any)["type"] != "thinking" {
		t.Fatalf("blocks = %#v", blocks)
	}
	if blocks[2].(map[string]any)["type"] != "tool_use" {
		t.Fatalf("tool block = %#v", blocks[2])
	}
}
