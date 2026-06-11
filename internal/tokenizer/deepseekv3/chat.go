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

import "strings"

const (
	bosToken              = "<｜begin▁of▁sentence｜>"
	eosToken              = "<｜end▁of▁sentence｜>"
	userToken             = "<｜User｜>"
	assistantToken        = "<｜Assistant｜>"
	toolCallsBeginToken   = "<｜tool▁calls▁begin｜>"
	toolCallsEndToken     = "<｜tool▁calls▁end｜>"
	toolCallBeginToken    = "<｜tool▁call▁begin｜>"
	toolCallEndToken      = "<｜tool▁call▁end｜>"
	toolOutputsBeginToken = "<｜tool▁outputs▁begin｜>"
	toolOutputsEndToken   = "<｜tool▁outputs▁end｜>"
	toolOutputBeginToken  = "<｜tool▁output▁begin｜>"
	toolOutputEndToken    = "<｜tool▁output▁end｜>"
	toolSepToken          = "<｜tool▁sep｜>"
)

func RenderChat(messages []Message, addGenerationPrompt bool) string {
	var systemParts []string
	for _, message := range messages {
		if message.Role == "system" {
			systemParts = append(systemParts, message.Content)
		}
	}
	var b strings.Builder
	b.WriteString(bosToken)
	b.WriteString(strings.Join(systemParts, "\n\n"))

	isTool := false
	outputFirst := true
	for _, message := range messages {
		switch message.Role {
		case "system":
			continue
		case "user":
			isTool = false
			b.WriteString(userToken)
			b.WriteString(message.Content)
		case "assistant":
			if len(message.ToolCalls) > 0 {
				isTool = false
				b.WriteString(assistantToken)
				b.WriteString(stripReasoningPrefix(message.Content))
				b.WriteString(toolCallsBeginToken)
				for i, tool := range message.ToolCalls {
					if i > 0 {
						b.WriteByte('\n')
					}
					typ := tool.Type
					if typ == "" {
						typ = "function"
					}
					b.WriteString(toolCallBeginToken)
					b.WriteString(typ)
					b.WriteString(toolSepToken)
					b.WriteString(tool.Name)
					b.WriteString("\n```json\n")
					b.WriteString(tool.Arguments)
					b.WriteString("\n```")
					b.WriteString(toolCallEndToken)
				}
				b.WriteString(toolCallsEndToken)
				b.WriteString(eosToken)
				continue
			}
			content := stripReasoningPrefix(message.Content)
			if isTool {
				b.WriteString(toolOutputsEndToken)
				b.WriteString(content)
				b.WriteString(eosToken)
				isTool = false
			} else {
				b.WriteString(assistantToken)
				b.WriteString(content)
				b.WriteString(eosToken)
			}
		case "tool":
			isTool = true
			if outputFirst {
				b.WriteString(toolOutputsBeginToken)
				outputFirst = false
			}
			b.WriteString(toolOutputBeginToken)
			b.WriteString(message.Content)
			b.WriteString(toolOutputEndToken)
		}
	}
	if isTool {
		b.WriteString(toolOutputsEndToken)
	}
	if addGenerationPrompt && !isTool {
		b.WriteString(assistantToken)
	}
	return b.String()
}

func stripReasoningPrefix(content string) string {
	if idx := strings.LastIndex(content, "</think>"); idx >= 0 {
		return content[idx+len("</think>"):]
	}
	return content
}
