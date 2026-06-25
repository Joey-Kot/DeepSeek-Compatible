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

package httpapi

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	anthropic "deepseek-responses-compatible/internal/adapters/anthropic/messages"
	gemini "deepseek-responses-compatible/internal/adapters/gemini/generate"
	"deepseek-responses-compatible/internal/adapters/openai/chat"
	"deepseek-responses-compatible/internal/adapters/openai/responses"
	"deepseek-responses-compatible/internal/adapters/openai/shared"
	"deepseek-responses-compatible/internal/config"
	"deepseek-responses-compatible/internal/debuglog"
	"deepseek-responses-compatible/internal/state"
)

type Upstream interface {
	Chat(ctx context.Context, payload shared.Map) (shared.Map, error)
	StreamChat(ctx context.Context, payload shared.Map, handle func(shared.Map) error) error
}

type Server struct {
	cfg       config.Config
	store     *state.Store
	upstream  Upstream
	chat      chat.Adapter
	responses responses.Adapter
	anthropic anthropic.Adapter
	gemini    gemini.Adapter
}

func New(cfg config.Config, upstream Upstream, store *state.Store) *Server {
	if store == nil {
		store = state.New()
	}
	return &Server{
		cfg:       cfg,
		store:     store,
		upstream:  upstream,
		chat:      chat.Adapter{DefaultModel: cfg.DefaultModel},
		responses: responses.Adapter{DefaultModel: cfg.DefaultModel, Store: store},
		anthropic: anthropic.Adapter{DefaultModel: cfg.DefaultModel},
		gemini:    gemini.Adapter{DefaultModel: cfg.DefaultModel},
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.cfg.DebugLogBody {
		debugWriter := newDebugResponseWriter(w)
		s.serveHTTP(debugWriter, r)
		log.Printf("debug body response method=%s path=%s status=%d body=%s", r.Method, r.URL.RequestURI(), debugWriter.statusCode(), debuglog.Body(debugWriter.bodyBytes()))
		return
	}
	s.serveHTTP(w, r)
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	s.setCommonHeaders(w)
	s.store.MaybePrune(time.Now(), s.cfg.StorePruneInterval)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.URL.Path == "/health" {
		writeJSON(w, http.StatusOK, shared.Map{"status": "ok"})
		return
	}
	if !s.authorize(w, r) {
		return
	}
	if r.URL.Path == "/health/memory" || r.URL.Path == "/healthz/memory" {
		s.handleMemoryHealth(w, r)
		return
	}
	if s.handleDebug(w, r) {
		return
	}

	path := strings.TrimRight(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}
	switch {
	case path == "/chat/completions":
		s.handleDeepSeekChatCompletions(w, r)
	case path == "/v1/messages" || path == "/v1/messages/count_tokens":
		s.handleAnthropicMessages(w, r, path)
	case strings.HasPrefix(path, "/v1beta/models/") || strings.HasPrefix(path, "/v1/models/"):
		if s.handleGeminiModels(w, r, path) {
			return
		}
		openAIError(w, http.StatusNotFound, "not found", "invalid_request_error", "")
	case r.Method == http.MethodGet && path == "/v1/models":
		s.handleModels(w, r)
	case path == "/v1/chat/completions":
		s.handleChatCompletions(w, r)
	case strings.HasPrefix(path, "/v1/chat/completions/"):
		s.handleStoredChatCompletion(w, r, strings.TrimPrefix(path, "/v1/chat/completions/"))
	case path == "/v1/responses":
		s.handleResponses(w, r)
	case path == "/v1/responses/input_tokens":
		s.handleInputTokens(w, r)
	case path == "/v1/responses/compact":
		s.handleCompact(w, r)
	case strings.HasPrefix(path, "/v1/responses/"):
		s.handleStoredResponse(w, r, strings.TrimPrefix(path, "/v1/responses/"))
	case path == "/v1/conversations":
		s.handleConversations(w, r)
	case strings.HasPrefix(path, "/v1/conversations/"):
		s.handleStoredConversation(w, r, strings.TrimPrefix(path, "/v1/conversations/"))
	default:
		openAIError(w, http.StatusNotFound, "not found", "invalid_request_error", "")
	}
}

func (s *Server) handleDeepSeekChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	payload, ok := s.readJSON(w, r, false)
	if !ok {
		return
	}
	if shared.BoolValue(payload["stream"]) {
		s.streamDeepSeekChatCompletion(w, r, payload)
		return
	}
	payload["stream"] = false
	completion, err := s.upstream.Chat(r.Context(), payload)
	if err != nil {
		s.upstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, completion)
}

func (s *Server) handleAnthropicMessages(w http.ResponseWriter, r *http.Request, path string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	payload, ok := s.readJSON(w, r, false)
	if !ok {
		return
	}
	if path == "/v1/messages/count_tokens" {
		result, err := anthropic.CountTokens(payload, s.cfg.DefaultModel)
		if err != nil {
			openAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "")
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	prepared, err := s.anthropic.BuildDeepSeekPayload(payload)
	if err != nil {
		openAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "")
		return
	}
	if shared.BoolValue(payload["stream"]) {
		s.streamAnthropicMessage(w, r, payload, prepared.ChatPayload)
		return
	}
	prepared.ChatPayload["stream"] = false
	completion, err := s.upstream.Chat(r.Context(), prepared.ChatPayload)
	if err != nil {
		s.upstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, anthropic.ResponseFromDeepSeek(completion, payload, s.cfg.DefaultModel))
}

func (s *Server) handleGeminiModels(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return true
	}
	model, action, ok := parseGeminiPath(path)
	if !ok {
		return false
	}
	payload, readOK := s.readJSON(w, r, false)
	if !readOK {
		return true
	}
	if action == "countTokens" {
		result, err := gemini.CountTokens(payload)
		if err != nil {
			openAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "")
			return true
		}
		writeJSON(w, http.StatusOK, result)
		return true
	}
	prepared, err := s.gemini.BuildDeepSeekPayload(model, payload)
	if err != nil {
		openAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "")
		return true
	}
	if action == "streamGenerateContent" {
		s.streamGeminiContent(w, r, model, prepared.ChatPayload)
		return true
	}
	if action != "generateContent" {
		return false
	}
	prepared.ChatPayload["stream"] = false
	completion, err := s.upstream.Chat(r.Context(), prepared.ChatPayload)
	if err != nil {
		s.upstreamError(w, err)
		return true
	}
	writeJSON(w, http.StatusOK, gemini.ResponseFromDeepSeek(completion, model))
	return true
}

func (s *Server) handleModels(w http.ResponseWriter, _ *http.Request) {
	data := []any{}
	for _, model := range s.cfg.ModelIDs {
		data = append(data, shared.Map{"id": model, "object": "model", "owned_by": "deepseek"})
	}
	writeJSON(w, http.StatusOK, shared.Map{"object": "list", "data": data})
}

func (s *Server) handleMemoryHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	storeStats := s.store.Stats()
	writeJSON(w, http.StatusOK, shared.Map{
		"alloc":        mem.Alloc,
		"heap_alloc":   mem.HeapAlloc,
		"heap_inuse":   mem.HeapInuse,
		"sys":          mem.Sys,
		"num_gc":       mem.NumGC,
		"goroutines":   runtime.NumGoroutine(),
		"store":        storeStats,
		"store_limits": shared.Map{"responses": s.cfg.StoreMaxResponses, "chat_completions": s.cfg.StoreMaxChatCompletions, "conversations": s.cfg.StoreMaxConversations, "ttl_seconds": s.cfg.StoreTTL.Seconds(), "prune_interval_seconds": s.cfg.StorePruneInterval.Seconds()},
	})
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		payload, ok := s.readJSON(w, r, false)
		if !ok {
			return
		}
		chatPayload, requestMessages, err := s.chat.BuildDeepSeekPayload(payload)
		if err != nil {
			openAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "")
			return
		}
		if shared.BoolValue(payload["stream"]) {
			s.streamChatCompletion(w, r, payload, chatPayload)
			return
		}
		chatPayload["stream"] = false
		completion, err := s.upstream.Chat(r.Context(), chatPayload)
		if err != nil {
			s.upstreamError(w, err)
			return
		}
		openAICompletion := chat.CompletionFromDeepSeek(completion, payload, s.cfg.DefaultModel)
		if shared.BoolValue(payload["store"]) {
			s.store.SaveChatCompletion(openAICompletion, chat.StoredMessages(requestMessages, openAICompletion, shared.StringValue(openAICompletion["id"])))
		}
		writeJSON(w, http.StatusOK, openAICompletion)
	case http.MethodGet:
		limit, order := paginationParams(r, 20, "asc")
		model := r.URL.Query().Get("model")
		items := s.store.ListChatCompletions(model, metadataFilters(r))
		writeJSON(w, http.StatusOK, shared.Paginate(items, r.URL.Query().Get("after"), limit, order))
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleStoredChatCompletion(w http.ResponseWriter, r *http.Request, rest string) {
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		openAIError(w, http.StatusNotFound, "not found", "invalid_request_error", "")
		return
	}
	if len(parts) == 2 && parts[1] == "messages" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		messages, ok := s.store.ChatCompletionMessagesFor(id)
		if !ok {
			openAIError(w, http.StatusNotFound, "Chat completion not found: "+id, "invalid_request_error", "")
			return
		}
		limit, order := paginationParams(r, 20, "asc")
		writeJSON(w, http.StatusOK, shared.Paginate(messages, r.URL.Query().Get("after"), limit, order))
		return
	}
	if len(parts) != 1 {
		openAIError(w, http.StatusNotFound, "not found", "invalid_request_error", "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		completion, ok := s.store.ChatCompletion(id)
		if !ok {
			openAIError(w, http.StatusNotFound, "Chat completion not found: "+id, "invalid_request_error", "")
			return
		}
		writeJSON(w, http.StatusOK, completion)
	case http.MethodPost:
		payload, ok := s.readJSON(w, r, false)
		if !ok {
			return
		}
		completion, ok := s.store.UpdateChatCompletion(id, payload["metadata"])
		if !ok {
			openAIError(w, http.StatusNotFound, "Chat completion not found: "+id, "invalid_request_error", "")
			return
		}
		writeJSON(w, http.StatusOK, completion)
	case http.MethodDelete:
		if !s.store.DeleteChatCompletion(id) {
			openAIError(w, http.StatusNotFound, "Chat completion not found: "+id, "invalid_request_error", "")
			return
		}
		writeJSON(w, http.StatusOK, shared.Map{"id": id, "object": "chat.completion.deleted", "deleted": true})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleResponses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	payload, ok := s.readJSON(w, r, false)
	if !ok {
		return
	}
	prepared, err := s.responses.Prepare(payload)
	if err != nil {
		openAIError(w, statusForLookupError(err), err.Error(), "invalid_request_error", "")
		return
	}
	chatPayload, toolNameMap := s.responses.BuildDeepSeekPayload(payload, prepared.Messages)
	if shared.BoolValue(payload["stream"]) {
		s.streamResponse(w, r, payload, prepared, chatPayload, toolNameMap)
		return
	}
	chatPayload["stream"] = false
	completion, err := s.upstream.Chat(r.Context(), chatPayload)
	if err != nil {
		s.upstreamError(w, err)
		return
	}
	outputItems, outputText, finishReason, _ := s.responses.OutputItemsFromChatCompletion(completion, toolNameMap)
	status, incompleteDetails := responses.StatusFromFinishReason(finishReason)
	responseID := shared.NewID("resp")
	response := s.responses.BaseResponse(payload, responseID, shared.NowSeconds(), status, outputItems, outputText, responses.ResponseUsageFromDeepSeek(completion["usage"]), incompleteDetails)
	s.store.SaveResponse(response, prepared.AllItems, outputItems, payload["store"] != false, prepared.ConversationID, prepared.InputItems)
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleStoredResponse(w http.ResponseWriter, r *http.Request, rest string) {
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		openAIError(w, http.StatusNotFound, "not found", "invalid_request_error", "")
		return
	}
	if len(parts) == 2 && parts[1] == "input_items" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		items, ok := s.store.ResponseInput(id)
		if !ok {
			openAIError(w, http.StatusNotFound, "Response not found: "+id, "invalid_request_error", "")
			return
		}
		limit, order := paginationParams(r, 20, "desc")
		writeJSON(w, http.StatusOK, shared.Paginate(items, r.URL.Query().Get("after"), limit, order))
		return
	}
	if len(parts) == 2 && parts[1] == "cancel" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		response, ok := s.store.UpdateResponse(id, func(item shared.Map) shared.Map {
			status := shared.StringValue(item["status"])
			if status == "queued" || status == "in_progress" {
				item["status"] = "cancelled"
				item["completed_at"] = shared.NowSeconds()
			}
			return item
		})
		if !ok {
			openAIError(w, http.StatusNotFound, "Response not found: "+id, "invalid_request_error", "")
			return
		}
		if shared.StringValue(response["status"]) != "cancelled" {
			openAIError(w, http.StatusBadRequest, "Only in-progress background responses can be cancelled", "invalid_request_error", "")
			return
		}
		writeJSON(w, http.StatusOK, response)
		return
	}
	if len(parts) != 1 {
		openAIError(w, http.StatusNotFound, "not found", "invalid_request_error", "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		response, ok := s.store.Response(id)
		if !ok {
			openAIError(w, http.StatusNotFound, "Response not found: "+id, "invalid_request_error", "")
			return
		}
		writeJSON(w, http.StatusOK, response)
	case http.MethodDelete:
		if !s.store.DeleteResponse(id) {
			openAIError(w, http.StatusNotFound, "Response not found: "+id, "invalid_request_error", "")
			return
		}
		writeJSON(w, http.StatusOK, shared.Map{"id": id, "object": "response", "deleted": true})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleInputTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	payload, ok := s.readJSON(w, r, false)
	if !ok {
		return
	}
	prepared, err := s.responses.Prepare(payload)
	if err != nil {
		openAIError(w, statusForLookupError(err), err.Error(), "invalid_request_error", "")
		return
	}
	writeJSON(w, http.StatusOK, shared.Map{"object": "response.input_tokens", "input_tokens": shared.EstimateTokensFromMessages(prepared.Messages)})
}

func (s *Server) handleCompact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	payload, ok := s.readJSON(w, r, false)
	if !ok {
		return
	}
	compactPayload := shared.CloneMap(payload)
	if shared.StringValue(compactPayload["instructions"]) == "" {
		compactPayload["instructions"] = "Compact the provided conversation into a concise context summary for future turns."
	}
	if compactPayload["text"] == nil {
		compactPayload["text"] = shared.Map{"format": shared.Map{"type": "text"}}
	}
	prepared, err := s.responses.Prepare(compactPayload)
	if err != nil {
		openAIError(w, statusForLookupError(err), err.Error(), "invalid_request_error", "")
		return
	}
	chatPayload, toolNameMap := s.responses.BuildDeepSeekPayload(compactPayload, prepared.Messages)
	chatPayload["stream"] = false
	completion, err := s.upstream.Chat(r.Context(), chatPayload)
	if err != nil {
		s.upstreamError(w, err)
		return
	}
	outputItems, _, finishReason, _ := s.responses.OutputItemsFromChatCompletion(completion, toolNameMap)
	status, _ := responses.StatusFromFinishReason(finishReason)
	writeJSON(w, http.StatusOK, shared.Map{
		"id":         shared.NewID("comp"),
		"created_at": shared.NowSeconds(),
		"object":     "response.compaction",
		"status":     status,
		"output":     shared.PublicItems(outputItems),
		"usage":      responses.ResponseUsageFromDeepSeek(completion["usage"]),
	})
}

func (s *Server) handleConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	payload, ok := s.readJSON(w, r, true)
	if !ok {
		return
	}
	id := shared.NewID("conv")
	conversation := shared.Map{"id": id, "object": "conversation", "created_at": shared.NowSeconds(), "metadata": metadataOrEmpty(payload["metadata"])}
	items, err := s.responses.NormalizeInputItems(payload["items"])
	if err != nil {
		openAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "")
		return
	}
	s.store.SaveConversation(conversation, items)
	writeJSON(w, http.StatusOK, conversation)
}

func (s *Server) handleStoredConversation(w http.ResponseWriter, r *http.Request, id string) {
	if id == "" || strings.Contains(id, "/") {
		openAIError(w, http.StatusNotFound, "not found", "invalid_request_error", "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		conversation, ok := s.store.Conversation(id)
		if !ok {
			openAIError(w, http.StatusNotFound, "Conversation not found: "+id, "invalid_request_error", "")
			return
		}
		writeJSON(w, http.StatusOK, conversation)
	case http.MethodPost, http.MethodPatch:
		payload, ok := s.readJSON(w, r, false)
		if !ok {
			return
		}
		conversation, ok := s.store.UpdateConversation(id, payload["metadata"])
		if !ok {
			openAIError(w, http.StatusNotFound, "Conversation not found: "+id, "invalid_request_error", "")
			return
		}
		writeJSON(w, http.StatusOK, conversation)
	case http.MethodDelete:
		if !s.store.DeleteConversation(id) {
			openAIError(w, http.StatusNotFound, "Conversation not found: "+id, "invalid_request_error", "")
			return
		}
		writeJSON(w, http.StatusOK, shared.Map{"id": id, "deleted": true, "object": "conversation.deleted"})
	default:
		methodNotAllowed(w)
	}
}

func paginationParams(r *http.Request, defaultLimit int, defaultOrder string) (int, string) {
	limit := defaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	order := r.URL.Query().Get("order")
	if order != "asc" && order != "desc" {
		order = defaultOrder
	}
	return limit, order
}

var metadataFilterRe = regexp.MustCompile(`^metadata\[([^\]]+)\]$`)

func metadataFilters(r *http.Request) map[string]string {
	out := map[string]string{}
	for key, values := range r.URL.Query() {
		match := metadataFilterRe.FindStringSubmatch(key)
		if len(match) == 2 && len(values) > 0 {
			out[match[1]] = values[0]
		}
	}
	return out
}

func metadataOrEmpty(value any) any {
	if value == nil {
		return shared.Map{}
	}
	return value
}

func parseGeminiPath(path string) (string, string, bool) {
	prefixes := []string{"/v1beta/models/", "/v1/models/"}
	rest := ""
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			rest = strings.TrimPrefix(path, prefix)
			break
		}
	}
	if rest == "" {
		return "", "", false
	}
	parts := strings.Split(rest, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
