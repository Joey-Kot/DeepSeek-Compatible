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

package deepseekv3

import (
	"strings"
	"testing"
)

func TestDefaultTokenizerCountsText(t *testing.T) {
	tok, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if got := tok.Count("Hello!"); got != 2 {
		t.Fatalf("Hello! tokens = %d", got)
	}
	if got := tok.Count("你好"); got <= 0 {
		t.Fatalf("Chinese tokens = %d", got)
	}
}

func TestDefaultTokenizerCountsAddedTokens(t *testing.T) {
	tok, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if got := tok.Count("<｜User｜>Hello!"); got != 3 {
		t.Fatalf("special token count = %d", got)
	}
}

func TestCountMessagesUsesDeepSeekChatTemplate(t *testing.T) {
	got := CountMessages([]Message{
		{Role: "system", Content: "Be brief."},
		{Role: "user", Content: "Hello!"},
	})
	if got <= CountText("Be brief.\nHello!") {
		t.Fatalf("chat template token count = %d", got)
	}
}

func TestRenderChatIncludesToolCalls(t *testing.T) {
	rendered := RenderChat([]Message{
		{Role: "assistant", Content: "Calling.", ToolCalls: []ToolCall{{Type: "function", Name: "get_weather", Arguments: `{"city":"Hangzhou"}`}}},
		{Role: "tool", Content: "Sunny"},
	}, true)
	for _, want := range []string{toolCallsBeginToken, toolCallBeginToken, toolOutputsBeginToken, toolOutputBeginToken, toolOutputsEndToken} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered chat missing %q: %s", want, rendered)
		}
	}
}
