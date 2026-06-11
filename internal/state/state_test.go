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

package state

import (
	"testing"

	"deepseek-responses-compatible/internal/adapters/openai/shared"
)

func TestRegisterItemsStoresClone(t *testing.T) {
	store := New()
	item := shared.Map{"id": "item_1", "content": "hello"}
	store.RegisterItems([]shared.Map{item})
	item["content"] = "mutated"

	stored, ok := store.Item("item_1")
	if !ok {
		t.Fatal("item was not stored")
	}
	if stored["content"] != "hello" {
		t.Fatalf("stored item was mutated: %#v", stored)
	}
}

func TestResponseLifecycle(t *testing.T) {
	store := New()
	response := shared.Map{"id": "resp_1", "status": "completed"}
	context := []shared.Map{{"id": "msg_in", "role": "user"}}
	output := []shared.Map{{"id": "msg_out", "role": "assistant"}}
	store.SaveResponse(response, context, output, true, "", nil)

	got, ok := store.Response("resp_1")
	if !ok || got["status"] != "completed" {
		t.Fatalf("response = %#v ok=%v", got, ok)
	}
	input, ok := store.ResponseInput("resp_1")
	if !ok || len(input) != 1 || input[0]["id"] != "msg_in" {
		t.Fatalf("input = %#v ok=%v", input, ok)
	}
	full, ok := store.ResponseContext("resp_1")
	if !ok || len(full) != 2 {
		t.Fatalf("context = %#v ok=%v", full, ok)
	}
	updated, ok := store.UpdateResponse("resp_1", func(item shared.Map) shared.Map {
		item["status"] = "cancelled"
		return item
	})
	if !ok || updated["status"] != "cancelled" {
		t.Fatalf("updated = %#v ok=%v", updated, ok)
	}
	if !store.DeleteResponse("resp_1") || store.DeleteResponse("resp_1") {
		t.Fatalf("delete response returned unexpected result")
	}
}

func TestUnstoredResponseStillKeepsContext(t *testing.T) {
	store := New()
	store.SaveResponse(shared.Map{"id": "resp_1"}, []shared.Map{{"id": "msg_in"}}, nil, false, "", nil)
	if _, ok := store.Response("resp_1"); ok {
		t.Fatal("response should not be stored when store=false")
	}
	if input, ok := store.ResponseInput("resp_1"); !ok || len(input) != 1 {
		t.Fatalf("input context missing: %#v ok=%v", input, ok)
	}
}

func TestConversationLifecycleAndResponseAppend(t *testing.T) {
	store := New()
	store.SaveConversation(shared.Map{"id": "conv_1", "metadata": shared.Map{"topic": "demo"}}, []shared.Map{{"id": "msg_1"}})
	if conv, ok := store.Conversation("conv_1"); !ok || conv["id"] != "conv_1" {
		t.Fatalf("conversation = %#v ok=%v", conv, ok)
	}
	updated, ok := store.UpdateConversation("conv_1", shared.Map{"topic": "updated"})
	if !ok || updated["metadata"].(map[string]any)["topic"] != "updated" {
		t.Fatalf("updated conversation = %#v ok=%v", updated, ok)
	}
	store.SaveResponse(shared.Map{"id": "resp_1"}, []shared.Map{}, []shared.Map{{"id": "msg_out"}}, true, "conv_1", []shared.Map{{"id": "msg_in"}})
	items, ok := store.ConversationItemsFor("conv_1")
	if !ok || len(items) != 3 {
		t.Fatalf("conversation items = %#v ok=%v", items, ok)
	}
	if !store.DeleteConversation("conv_1") || store.DeleteConversation("conv_1") {
		t.Fatalf("delete conversation returned unexpected result")
	}
}

func TestChatCompletionLifecycleAndFiltering(t *testing.T) {
	store := New()
	store.SaveChatCompletion(
		shared.Map{"id": "chat_2", "created": 2, "model": "deepseek-v4-pro", "metadata": shared.Map{"topic": "skip"}},
		[]shared.Map{{"id": "msg_2"}},
	)
	store.SaveChatCompletion(
		shared.Map{"id": "chat_1", "created": 1, "model": "deepseek-v4-pro", "metadata": shared.Map{"topic": "demo"}},
		[]shared.Map{{"id": "msg_1"}},
	)
	items := store.ListChatCompletions("deepseek-v4-pro", map[string]string{"topic": "demo"})
	if len(items) != 1 || items[0]["id"] != "chat_1" {
		t.Fatalf("filtered items = %#v", items)
	}
	updated, ok := store.UpdateChatCompletion("chat_1", shared.Map{"topic": "updated"})
	if !ok || updated["metadata"].(map[string]any)["topic"] != "updated" {
		t.Fatalf("updated chat = %#v ok=%v", updated, ok)
	}
	messages, ok := store.ChatCompletionMessagesFor("chat_1")
	if !ok || len(messages) != 1 || messages[0]["id"] != "msg_1" {
		t.Fatalf("messages = %#v ok=%v", messages, ok)
	}
	if !store.DeleteChatCompletion("chat_1") || store.DeleteChatCompletion("chat_1") {
		t.Fatalf("delete chat returned unexpected result")
	}
}
