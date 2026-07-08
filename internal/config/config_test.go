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

package config

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":8080" {
		t.Fatalf("Listen = %q", cfg.Listen)
	}
	if cfg.DeepSeekBaseURL != DefaultDeepSeekBaseURL {
		t.Fatalf("DeepSeekBaseURL = %q", cfg.DeepSeekBaseURL)
	}
	if cfg.DefaultModel != DefaultModel {
		t.Fatalf("DefaultModel = %q", cfg.DefaultModel)
	}
	if !reflect.DeepEqual(cfg.ModelIDs, []string{DefaultModel}) {
		t.Fatalf("ModelIDs = %#v", cfg.ModelIDs)
	}
	if cfg.DeepSeekHTTPTimeout != 120*time.Second {
		t.Fatalf("DeepSeekHTTPTimeout = %s", cfg.DeepSeekHTTPTimeout)
	}
	if cfg.DeepSeekMaxIdleConns != 200 {
		t.Fatalf("DeepSeekMaxIdleConns = %d", cfg.DeepSeekMaxIdleConns)
	}
	if cfg.DeepSeekMaxIdleConnsPerHost != 100 {
		t.Fatalf("DeepSeekMaxIdleConnsPerHost = %d", cfg.DeepSeekMaxIdleConnsPerHost)
	}
	if cfg.DeepSeekMaxConnsPerHost != 0 {
		t.Fatalf("DeepSeekMaxConnsPerHost = %d", cfg.DeepSeekMaxConnsPerHost)
	}
	if cfg.DeepSeekMaxResponseBodyBytes != 32<<20 {
		t.Fatalf("DeepSeekMaxResponseBodyBytes = %d", cfg.DeepSeekMaxResponseBodyBytes)
	}
	if cfg.StoreMaxResponses != 1000 {
		t.Fatalf("StoreMaxResponses = %d", cfg.StoreMaxResponses)
	}
	if cfg.StoreMaxChatCompletions != 1000 {
		t.Fatalf("StoreMaxChatCompletions = %d", cfg.StoreMaxChatCompletions)
	}
	if cfg.StoreMaxConversations != 1000 {
		t.Fatalf("StoreMaxConversations = %d", cfg.StoreMaxConversations)
	}
	if cfg.StoreTTL != time.Hour {
		t.Fatalf("StoreTTL = %s", cfg.StoreTTL)
	}
	if cfg.StorePruneInterval != time.Minute {
		t.Fatalf("StorePruneInterval = %s", cfg.StorePruneInterval)
	}
	if cfg.MaxRequestBodyBytes != 16<<20 {
		t.Fatalf("MaxRequestBodyBytes = %d", cfg.MaxRequestBodyBytes)
	}
	if cfg.ReadHeaderTimeout != 10*time.Second {
		t.Fatalf("ReadHeaderTimeout = %s", cfg.ReadHeaderTimeout)
	}
	if cfg.IdleTimeout != 120*time.Second {
		t.Fatalf("IdleTimeout = %s", cfg.IdleTimeout)
	}
	if !cfg.VerifySSL {
		t.Fatal("VerifySSL should default to true")
	}
}

func TestParseCommandLineFlags(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Parse([]string{
		"--listen", "127.0.0.1:9999",
		"--api-token", "sk-a, sk-b ,,",
		"--deepseek-api-key", "sk-upstream",
		"--deepseek-base-url", "https://example.test/v1",
		"--deepseek-model", "deepseek-v4-flash",
		"--deepseek-models", "deepseek-v4-pro,deepseek-v4-flash",
		"--deepseek-http-timeout", "2.5",
		"--deepseek-max-idle-conns", "300",
		"--deepseek-max-idle-conns-per-host", "150",
		"--deepseek-max-conns-per-host", "80",
		"--deepseek-max-response-body-bytes", "2048",
		"--store-max-responses", "11",
		"--store-max-chat-completions", "12",
		"--store-max-conversations", "13",
		"--store-ttl", "600",
		"--store-prune-interval", "30",
		"--max-request-body-bytes", "1024",
		"--read-header-timeout", "3.5",
		"--idle-timeout", "45",
		"--debug-pprof=true",
		"--debug-log-body=true",
		"--verify-ssl=false",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.APITokens, []string{"sk-a", "sk-b"}) {
		t.Fatalf("APITokens = %#v", cfg.APITokens)
	}
	if cfg.DeepSeekAPIKey != "sk-upstream" {
		t.Fatalf("DeepSeekAPIKey = %q", cfg.DeepSeekAPIKey)
	}
	if !reflect.DeepEqual(cfg.ModelIDs, []string{"deepseek-v4-pro", "deepseek-v4-flash"}) {
		t.Fatalf("ModelIDs = %#v", cfg.ModelIDs)
	}
	if cfg.DeepSeekHTTPTimeout != 2500*time.Millisecond {
		t.Fatalf("DeepSeekHTTPTimeout = %s", cfg.DeepSeekHTTPTimeout)
	}
	if cfg.DeepSeekMaxIdleConns != 300 {
		t.Fatalf("DeepSeekMaxIdleConns = %d", cfg.DeepSeekMaxIdleConns)
	}
	if cfg.DeepSeekMaxIdleConnsPerHost != 150 {
		t.Fatalf("DeepSeekMaxIdleConnsPerHost = %d", cfg.DeepSeekMaxIdleConnsPerHost)
	}
	if cfg.DeepSeekMaxConnsPerHost != 80 {
		t.Fatalf("DeepSeekMaxConnsPerHost = %d", cfg.DeepSeekMaxConnsPerHost)
	}
	if cfg.DeepSeekMaxResponseBodyBytes != 2048 {
		t.Fatalf("DeepSeekMaxResponseBodyBytes = %d", cfg.DeepSeekMaxResponseBodyBytes)
	}
	if cfg.StoreMaxResponses != 11 {
		t.Fatalf("StoreMaxResponses = %d", cfg.StoreMaxResponses)
	}
	if cfg.StoreMaxChatCompletions != 12 {
		t.Fatalf("StoreMaxChatCompletions = %d", cfg.StoreMaxChatCompletions)
	}
	if cfg.StoreMaxConversations != 13 {
		t.Fatalf("StoreMaxConversations = %d", cfg.StoreMaxConversations)
	}
	if cfg.StoreTTL != 10*time.Minute {
		t.Fatalf("StoreTTL = %s", cfg.StoreTTL)
	}
	if cfg.StorePruneInterval != 30*time.Second {
		t.Fatalf("StorePruneInterval = %s", cfg.StorePruneInterval)
	}
	if cfg.MaxRequestBodyBytes != 1024 {
		t.Fatalf("MaxRequestBodyBytes = %d", cfg.MaxRequestBodyBytes)
	}
	if cfg.ReadHeaderTimeout != 3500*time.Millisecond {
		t.Fatalf("ReadHeaderTimeout = %s", cfg.ReadHeaderTimeout)
	}
	if cfg.IdleTimeout != 45*time.Second {
		t.Fatalf("IdleTimeout = %s", cfg.IdleTimeout)
	}
	if !cfg.DebugLogBody {
		t.Fatalf("boolean flags were not parsed: %#v", cfg)
	}
	if !cfg.DebugPprof {
		t.Fatalf("DebugPprof = %t", cfg.DebugPprof)
	}
	if cfg.VerifySSL {
		t.Fatalf("VerifySSL = %t", cfg.VerifySSL)
	}
}

func TestParseEnvironmentVariables(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("LISTEN", "127.0.0.1:9999")
	t.Setenv("API_TOKEN", "sk-a, sk-b ,,")
	t.Setenv("DEEPSEEK_API_KEY", "sk-upstream")
	t.Setenv("DEEPSEEK_BASE_URL", "https://example.test/v1")
	t.Setenv("DEEPSEEK_MODEL", "deepseek-v4-flash")
	t.Setenv("DEEPSEEK_MODELS", "deepseek-v4-pro,deepseek-v4-flash")
	t.Setenv("DEEPSEEK_HTTP_TIMEOUT", "2.5")
	t.Setenv("DEEPSEEK_MAX_IDLE_CONNS", "300")
	t.Setenv("DEEPSEEK_MAX_IDLE_CONNS_PER_HOST", "150")
	t.Setenv("DEEPSEEK_MAX_CONNS_PER_HOST", "80")
	t.Setenv("DEEPSEEK_MAX_RESPONSE_BODY_BYTES", "2048")
	t.Setenv("STORE_MAX_RESPONSES", "11")
	t.Setenv("STORE_MAX_CHAT_COMPLETIONS", "12")
	t.Setenv("STORE_MAX_CONVERSATIONS", "13")
	t.Setenv("STORE_TTL", "600")
	t.Setenv("STORE_PRUNE_INTERVAL", "30")
	t.Setenv("MAX_REQUEST_BODY_BYTES", "1024")
	t.Setenv("READ_HEADER_TIMEOUT", "3.5")
	t.Setenv("IDLE_TIMEOUT", "45")
	t.Setenv("DEBUG_PPROF", "true")
	t.Setenv("DEBUG_LOG_BODY", "true")
	t.Setenv("VERIFY_SSL", "false")

	cfg, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "127.0.0.1:9999" {
		t.Fatalf("Listen = %q", cfg.Listen)
	}
	if !reflect.DeepEqual(cfg.APITokens, []string{"sk-a", "sk-b"}) {
		t.Fatalf("APITokens = %#v", cfg.APITokens)
	}
	if cfg.DeepSeekAPIKey != "sk-upstream" {
		t.Fatalf("DeepSeekAPIKey = %q", cfg.DeepSeekAPIKey)
	}
	if cfg.DeepSeekBaseURL != "https://example.test/v1" {
		t.Fatalf("DeepSeekBaseURL = %q", cfg.DeepSeekBaseURL)
	}
	if cfg.DefaultModel != "deepseek-v4-flash" {
		t.Fatalf("DefaultModel = %q", cfg.DefaultModel)
	}
	if !reflect.DeepEqual(cfg.ModelIDs, []string{"deepseek-v4-pro", "deepseek-v4-flash"}) {
		t.Fatalf("ModelIDs = %#v", cfg.ModelIDs)
	}
	if cfg.DeepSeekHTTPTimeout != 2500*time.Millisecond {
		t.Fatalf("DeepSeekHTTPTimeout = %s", cfg.DeepSeekHTTPTimeout)
	}
	if cfg.DeepSeekMaxIdleConns != 300 {
		t.Fatalf("DeepSeekMaxIdleConns = %d", cfg.DeepSeekMaxIdleConns)
	}
	if cfg.DeepSeekMaxIdleConnsPerHost != 150 {
		t.Fatalf("DeepSeekMaxIdleConnsPerHost = %d", cfg.DeepSeekMaxIdleConnsPerHost)
	}
	if cfg.DeepSeekMaxConnsPerHost != 80 {
		t.Fatalf("DeepSeekMaxConnsPerHost = %d", cfg.DeepSeekMaxConnsPerHost)
	}
	if cfg.DeepSeekMaxResponseBodyBytes != 2048 {
		t.Fatalf("DeepSeekMaxResponseBodyBytes = %d", cfg.DeepSeekMaxResponseBodyBytes)
	}
	if cfg.StoreMaxResponses != 11 {
		t.Fatalf("StoreMaxResponses = %d", cfg.StoreMaxResponses)
	}
	if cfg.StoreMaxChatCompletions != 12 {
		t.Fatalf("StoreMaxChatCompletions = %d", cfg.StoreMaxChatCompletions)
	}
	if cfg.StoreMaxConversations != 13 {
		t.Fatalf("StoreMaxConversations = %d", cfg.StoreMaxConversations)
	}
	if cfg.StoreTTL != 10*time.Minute {
		t.Fatalf("StoreTTL = %s", cfg.StoreTTL)
	}
	if cfg.StorePruneInterval != 30*time.Second {
		t.Fatalf("StorePruneInterval = %s", cfg.StorePruneInterval)
	}
	if cfg.MaxRequestBodyBytes != 1024 {
		t.Fatalf("MaxRequestBodyBytes = %d", cfg.MaxRequestBodyBytes)
	}
	if cfg.ReadHeaderTimeout != 3500*time.Millisecond {
		t.Fatalf("ReadHeaderTimeout = %s", cfg.ReadHeaderTimeout)
	}
	if cfg.IdleTimeout != 45*time.Second {
		t.Fatalf("IdleTimeout = %s", cfg.IdleTimeout)
	}
	if !cfg.DebugLogBody {
		t.Fatalf("DebugLogBody = %t", cfg.DebugLogBody)
	}
	if !cfg.DebugPprof {
		t.Fatalf("DebugPprof = %t", cfg.DebugPprof)
	}
	if cfg.VerifySSL {
		t.Fatalf("VerifySSL = %t", cfg.VerifySSL)
	}
}

func TestParseCommandLineFlagsTakePrecedenceOverEnvironmentVariables(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("LISTEN", "127.0.0.1:9999")
	t.Setenv("DEEPSEEK_HTTP_TIMEOUT", "0")
	t.Setenv("VERIFY_SSL", "false")

	cfg, err := Parse([]string{
		"--listen", "127.0.0.1:8081",
		"--deepseek-http-timeout", "2",
		"--verify-ssl=true",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "127.0.0.1:8081" {
		t.Fatalf("Listen = %q", cfg.Listen)
	}
	if cfg.DeepSeekHTTPTimeout != 2*time.Second {
		t.Fatalf("DeepSeekHTTPTimeout = %s", cfg.DeepSeekHTTPTimeout)
	}
	if !cfg.VerifySSL {
		t.Fatal("VerifySSL should come from command-line flag")
	}
}

func TestParsePrependsDefaultModelWhenMissingFromModelList(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Parse([]string{"--deepseek-model", "deepseek-main", "--deepseek-models", "deepseek-alt"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.ModelIDs, []string{"deepseek-main", "deepseek-alt"}) {
		t.Fatalf("ModelIDs = %#v", cfg.ModelIDs)
	}
}

func TestParseRejectsNonPositiveTimeout(t *testing.T) {
	clearConfigEnv(t)

	if _, err := Parse([]string{"--deepseek-http-timeout", "0"}); err == nil {
		t.Fatal("expected error for zero timeout")
	}
}

func TestParseRejectsInvalidConnectionLimits(t *testing.T) {
	clearConfigEnv(t)

	for _, flag := range []string{"--deepseek-max-idle-conns", "--deepseek-max-idle-conns-per-host", "--deepseek-max-conns-per-host", "--deepseek-max-response-body-bytes"} {
		if _, err := Parse([]string{flag, "-1"}); err == nil {
			t.Fatalf("expected error for %s", flag)
		}
	}
}

func TestParseRejectsInvalidStoreLimits(t *testing.T) {
	clearConfigEnv(t)

	for _, flag := range []string{"--store-max-responses", "--store-max-chat-completions", "--store-max-conversations", "--store-ttl", "--store-prune-interval", "--max-request-body-bytes"} {
		if _, err := Parse([]string{flag, "-1"}); err == nil {
			t.Fatalf("expected error for %s", flag)
		}
	}
}

func TestParseRejectsNonPositiveServerTimeouts(t *testing.T) {
	clearConfigEnv(t)

	for _, flag := range []string{"--read-header-timeout", "--idle-timeout"} {
		if _, err := Parse([]string{flag, "0"}); err == nil {
			t.Fatalf("expected error for %s", flag)
		}
	}
}

func TestParseRejectsInvalidEnvironmentVariable(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DEEPSEEK_HTTP_TIMEOUT", "not-a-number")

	if _, err := Parse(nil); err == nil || !strings.Contains(err.Error(), "DEEPSEEK_HTTP_TIMEOUT") {
		t.Fatalf("expected DEEPSEEK_HTTP_TIMEOUT error, got %v", err)
	}
}

func TestUsageIncludesCurrentFlagsAndProtocolSummary(t *testing.T) {
	clearConfigEnv(t)

	var out bytes.Buffer
	cfg := defaultConfig()
	var flags parseFlags
	fs := newFlagSet(&cfg, &flags)
	fs.SetOutput(&out)

	usage(fs)

	text := out.String()
	for _, want := range []string{
		"Usage:",
		"deepseek-compatible [flags]",
		"--deepseek-max-response-body-bytes",
		"--store-max-chat-completions",
		"--store-prune-interval",
		"--max-request-body-bytes",
		"--debug-pprof",
		"Command-line flags take precedence",
		"docker.env.example",
		"OpenAI Responses",
		"Gemini Generate Content",
		"/health/memory",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("usage output missing %q:\n%s", want, text)
		}
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, envFlag := range envFlags {
		t.Setenv(envFlag.Env, "")
	}
}
