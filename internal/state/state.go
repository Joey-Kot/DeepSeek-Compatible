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
	"sync"
	"time"

	"deepseek-responses-compatible/internal/adapters/openai/shared"
)

type Store struct {
	mu sync.RWMutex

	Responses              map[string]shared.Map
	ResponseInputItems     map[string][]shared.Map
	ResponseContextItems   map[string][]shared.Map
	ChatCompletions        map[string]shared.Map
	ChatCompletionMessages map[string][]shared.Map
	Conversations          map[string]shared.Map
	ConversationItems      map[string][]shared.Map
	ItemsByID              map[string]shared.Map

	limits                 Limits
	responseOrder          []string
	chatCompletionOrder    []string
	conversationOrder      []string
	responseAccessedAt     map[string]time.Time
	chatAccessedAt         map[string]time.Time
	conversationAccessedAt map[string]time.Time
	lastPrunedAt           time.Time
}

type Limits struct {
	MaxResponses       int
	MaxChatCompletions int
	MaxConversations   int
	TTL                time.Duration
}

type Stats struct {
	Responses              int `json:"responses"`
	ResponseInputItems     int `json:"response_input_items"`
	ResponseContextItems   int `json:"response_context_items"`
	ChatCompletions        int `json:"chat_completions"`
	ChatCompletionMessages int `json:"chat_completion_messages"`
	Conversations          int `json:"conversations"`
	ConversationItems      int `json:"conversation_items"`
	Items                  int `json:"items"`
}

func New() *Store {
	return NewWithLimits(Limits{})
}

func NewWithLimits(limits Limits) *Store {
	return &Store{
		Responses:              map[string]shared.Map{},
		ResponseInputItems:     map[string][]shared.Map{},
		ResponseContextItems:   map[string][]shared.Map{},
		ChatCompletions:        map[string]shared.Map{},
		ChatCompletionMessages: map[string][]shared.Map{},
		Conversations:          map[string]shared.Map{},
		ConversationItems:      map[string][]shared.Map{},
		ItemsByID:              map[string]shared.Map{},
		limits:                 limits,
		responseAccessedAt:     map[string]time.Time{},
		chatAccessedAt:         map[string]time.Time{},
		conversationAccessedAt: map[string]time.Time{},
	}
}

func (s *Store) RegisterItems(items []shared.Map) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registerItemsLocked(items)
}

func (s *Store) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Stats{
		Responses:              len(s.Responses),
		ResponseInputItems:     len(s.ResponseInputItems),
		ResponseContextItems:   len(s.ResponseContextItems),
		ChatCompletions:        len(s.ChatCompletions),
		ChatCompletionMessages: len(s.ChatCompletionMessages),
		Conversations:          len(s.Conversations),
		ConversationItems:      len(s.ConversationItems),
		Items:                  len(s.ItemsByID),
	}
}

func (s *Store) PruneExpired(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneExpiredLocked(now)
}

func (s *Store) MaybePrune(now time.Time, interval time.Duration) {
	if s.limits.TTL <= 0 || interval <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.lastPrunedAt.IsZero() && now.Sub(s.lastPrunedAt) < interval {
		return
	}
	s.lastPrunedAt = now
	s.pruneExpiredLocked(now)
}

func (s *Store) pruneExpiredLocked(now time.Time) {
	if s.limits.TTL <= 0 {
		return
	}
	cutoff := now.Add(-s.limits.TTL)
	for id, accessedAt := range s.responseAccessedAt {
		if !accessedAt.After(cutoff) {
			s.deleteResponseLocked(id)
		}
	}
	for id, accessedAt := range s.chatAccessedAt {
		if !accessedAt.After(cutoff) {
			s.deleteChatCompletionLocked(id)
		}
	}
	for id, accessedAt := range s.conversationAccessedAt {
		if !accessedAt.After(cutoff) {
			s.deleteConversationLocked(id)
		}
	}
}

func (s *Store) registerItemsLocked(items []shared.Map) {
	for _, item := range items {
		id := shared.StringValue(item["id"])
		if id != "" {
			s.ItemsByID[id] = shared.CloneMap(item)
		}
	}
}

func (s *Store) Item(id string) (shared.Map, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.ItemsByID[id]
	return shared.CloneMap(item), ok
}

func (s *Store) Response(id string) (shared.Map, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.Responses[id]
	if ok {
		s.touchResponseLocked(id)
	}
	return shared.CloneMap(item), ok
}

func (s *Store) SaveResponse(response shared.Map, contextItems, outputItems []shared.Map, store bool, conversationID string, currentInputItems []shared.Map) {
	s.mu.Lock()
	defer s.mu.Unlock()
	responseID := shared.StringValue(response["id"])
	if store {
		full := append(shared.CloneSlice(contextItems), shared.CloneSlice(outputItems)...)
		s.ResponseContextItems[responseID] = full
		s.ResponseInputItems[responseID] = shared.CloneSlice(contextItems)
		s.registerItemsLocked(contextItems)
		s.registerItemsLocked(outputItems)
		s.Responses[responseID] = shared.CloneMap(response)
		s.responseAccessedAt[responseID] = time.Now()
		s.responseOrder = rememberID(s.responseOrder, responseID)
		s.evictResponsesLocked()
	}
	if conversationID != "" {
		items := s.ConversationItems[conversationID]
		items = append(items, shared.CloneSlice(currentInputItems)...)
		items = append(items, shared.CloneSlice(outputItems)...)
		s.ConversationItems[conversationID] = items
		s.registerItemsLocked(items)
	}
}

func (s *Store) DeleteResponse(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteResponseLocked(id)
}

func (s *Store) deleteResponseLocked(id string) bool {
	if _, ok := s.Responses[id]; !ok {
		return false
	}
	items := append(shared.CloneSlice(s.ResponseInputItems[id]), s.ResponseContextItems[id]...)
	delete(s.Responses, id)
	delete(s.ResponseInputItems, id)
	delete(s.ResponseContextItems, id)
	delete(s.responseAccessedAt, id)
	s.responseOrder = removeID(s.responseOrder, id)
	s.deleteUnreferencedItemsLocked(items)
	return true
}

func (s *Store) ResponseInput(id string) ([]shared.Map, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, ok := s.ResponseInputItems[id]
	if ok {
		s.touchResponseLocked(id)
	}
	return shared.CloneSlice(items), ok
}

func (s *Store) ResponseContext(id string) ([]shared.Map, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, ok := s.ResponseContextItems[id]
	if ok {
		s.touchResponseLocked(id)
	}
	return shared.CloneSlice(items), ok
}

func (s *Store) UpdateResponse(id string, fn func(shared.Map) shared.Map) (shared.Map, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.Responses[id]
	if !ok {
		return nil, false
	}
	s.touchResponseLocked(id)
	updated := fn(shared.CloneMap(item))
	s.Responses[id] = shared.CloneMap(updated)
	return shared.CloneMap(updated), true
}

func (s *Store) SaveChatCompletion(completion shared.Map, messages []shared.Map) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := shared.StringValue(completion["id"])
	s.ChatCompletions[id] = shared.CloneMap(completion)
	s.ChatCompletionMessages[id] = shared.CloneSlice(messages)
	s.chatAccessedAt[id] = time.Now()
	s.chatCompletionOrder = rememberID(s.chatCompletionOrder, id)
	s.evictChatCompletionsLocked()
}

func (s *Store) ChatCompletion(id string) (shared.Map, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.ChatCompletions[id]
	if ok {
		s.touchChatCompletionLocked(id)
	}
	return shared.CloneMap(item), ok
}

func (s *Store) ChatCompletionMessagesFor(id string) ([]shared.Map, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.ChatCompletions[id]; !ok {
		return nil, false
	}
	s.touchChatCompletionLocked(id)
	return shared.CloneSlice(s.ChatCompletionMessages[id]), true
}

func (s *Store) ListChatCompletions(model string, metadata map[string]string) []shared.Map {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []shared.Map{}
	for _, completion := range s.ChatCompletions {
		if model != "" && shared.StringValue(completion["model"]) != model {
			continue
		}
		if !matchesMetadata(completion, metadata) {
			continue
		}
		out = append(out, shared.CloneMap(completion))
	}
	shared.SortByCreatedThenID(out)
	return out
}

func (s *Store) UpdateChatCompletion(id string, metadata any) (shared.Map, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	completion, ok := s.ChatCompletions[id]
	if !ok {
		return nil, false
	}
	s.touchChatCompletionLocked(id)
	completion = shared.CloneMap(completion)
	if metadata == nil {
		completion["metadata"] = shared.Map{}
	} else {
		completion["metadata"] = metadata
	}
	s.ChatCompletions[id] = shared.CloneMap(completion)
	return shared.CloneMap(completion), true
}

func (s *Store) DeleteChatCompletion(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteChatCompletionLocked(id)
}

func (s *Store) deleteChatCompletionLocked(id string) bool {
	if _, ok := s.ChatCompletions[id]; !ok {
		return false
	}
	delete(s.ChatCompletions, id)
	delete(s.ChatCompletionMessages, id)
	delete(s.chatAccessedAt, id)
	s.chatCompletionOrder = removeID(s.chatCompletionOrder, id)
	return true
}

func (s *Store) SaveConversation(conversation shared.Map, items []shared.Map) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := shared.StringValue(conversation["id"])
	s.Conversations[id] = shared.CloneMap(conversation)
	s.ConversationItems[id] = shared.CloneSlice(items)
	s.registerItemsLocked(items)
	s.conversationAccessedAt[id] = time.Now()
	s.conversationOrder = rememberID(s.conversationOrder, id)
	s.evictConversationsLocked()
}

func (s *Store) Conversation(id string) (shared.Map, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.Conversations[id]
	if ok {
		s.touchConversationLocked(id)
	}
	return shared.CloneMap(item), ok
}

func (s *Store) ConversationItemsFor(id string) ([]shared.Map, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Conversations[id]; !ok {
		return nil, false
	}
	s.touchConversationLocked(id)
	return shared.CloneSlice(s.ConversationItems[id]), true
}

func (s *Store) UpdateConversation(id string, metadata any) (shared.Map, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	conversation, ok := s.Conversations[id]
	if !ok {
		return nil, false
	}
	s.touchConversationLocked(id)
	conversation = shared.CloneMap(conversation)
	if metadata == nil {
		conversation["metadata"] = shared.Map{}
	} else {
		conversation["metadata"] = metadata
	}
	s.Conversations[id] = shared.CloneMap(conversation)
	return shared.CloneMap(conversation), true
}

func (s *Store) DeleteConversation(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteConversationLocked(id)
}

func (s *Store) deleteConversationLocked(id string) bool {
	if _, ok := s.Conversations[id]; !ok {
		return false
	}
	items := shared.CloneSlice(s.ConversationItems[id])
	delete(s.Conversations, id)
	delete(s.ConversationItems, id)
	delete(s.conversationAccessedAt, id)
	s.conversationOrder = removeID(s.conversationOrder, id)
	s.deleteUnreferencedItemsLocked(items)
	return true
}

func (s *Store) touchResponseLocked(id string) {
	if _, ok := s.responseAccessedAt[id]; ok {
		s.responseAccessedAt[id] = time.Now()
	}
}

func (s *Store) touchChatCompletionLocked(id string) {
	if _, ok := s.chatAccessedAt[id]; ok {
		s.chatAccessedAt[id] = time.Now()
	}
}

func (s *Store) touchConversationLocked(id string) {
	if _, ok := s.conversationAccessedAt[id]; ok {
		s.conversationAccessedAt[id] = time.Now()
	}
}

func (s *Store) evictResponsesLocked() {
	for s.limits.MaxResponses > 0 && len(s.Responses) > s.limits.MaxResponses && len(s.responseOrder) > 0 {
		id := s.responseOrder[0]
		s.deleteResponseLocked(id)
	}
}

func (s *Store) evictChatCompletionsLocked() {
	for s.limits.MaxChatCompletions > 0 && len(s.ChatCompletions) > s.limits.MaxChatCompletions && len(s.chatCompletionOrder) > 0 {
		id := s.chatCompletionOrder[0]
		s.deleteChatCompletionLocked(id)
	}
}

func (s *Store) evictConversationsLocked() {
	for s.limits.MaxConversations > 0 && len(s.Conversations) > s.limits.MaxConversations && len(s.conversationOrder) > 0 {
		id := s.conversationOrder[0]
		s.deleteConversationLocked(id)
	}
}

func (s *Store) deleteUnreferencedItemsLocked(items []shared.Map) {
	seen := map[string]bool{}
	for _, item := range items {
		id := shared.StringValue(item["id"])
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		if s.itemReferencedLocked(id) {
			continue
		}
		delete(s.ItemsByID, id)
	}
}

func (s *Store) itemReferencedLocked(id string) bool {
	for _, items := range s.ResponseInputItems {
		if sliceHasItemID(items, id) {
			return true
		}
	}
	for _, items := range s.ResponseContextItems {
		if sliceHasItemID(items, id) {
			return true
		}
	}
	for _, items := range s.ConversationItems {
		if sliceHasItemID(items, id) {
			return true
		}
	}
	return false
}

func sliceHasItemID(items []shared.Map, id string) bool {
	for _, item := range items {
		if shared.StringValue(item["id"]) == id {
			return true
		}
	}
	return false
}

func rememberID(order []string, id string) []string {
	if id == "" {
		return order
	}
	for _, existing := range order {
		if existing == id {
			return order
		}
	}
	return append(order, id)
}

func removeID(order []string, id string) []string {
	for i, existing := range order {
		if existing == id {
			return append(order[:i], order[i+1:]...)
		}
	}
	return order
}

func matchesMetadata(item shared.Map, filters map[string]string) bool {
	if len(filters) == 0 {
		return true
	}
	metadata, ok := item["metadata"].(map[string]any)
	if !ok {
		return false
	}
	for key, value := range filters {
		if shared.StringValue(metadata[key]) != value {
			return false
		}
	}
	return true
}
