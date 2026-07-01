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
	"flag"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultDeepSeekBaseURL = "https://api.deepseek.com"
	DefaultModel           = "deepseek-v4-pro"
)

type Config struct {
	Listen                       string
	APITokens                    []string
	DeepSeekAPIKey               string
	DeepSeekBaseURL              string
	DefaultModel                 string
	ModelIDs                     []string
	DeepSeekHTTPTimeout          time.Duration
	DeepSeekMaxIdleConns         int
	DeepSeekMaxIdleConnsPerHost  int
	DeepSeekMaxConnsPerHost      int
	DeepSeekMaxResponseBodyBytes int64
	StoreMaxResponses            int
	StoreMaxChatCompletions      int
	StoreMaxConversations        int
	StoreTTL                     time.Duration
	StorePruneInterval           time.Duration
	MaxRequestBodyBytes          int64
	ReadHeaderTimeout            time.Duration
	IdleTimeout                  time.Duration
	DebugPprof                   bool
	DebugLogBody                 bool
	VerifySSL                    bool
}

type parseFlags struct {
	apiTokenCSV               string
	modelCSV                  string
	timeoutSeconds            float64
	storeTTLSeconds           float64
	storePruneIntervalSeconds float64
	readHeaderTimeoutSeconds  float64
	idleTimeoutSeconds        float64
}

func Parse(args []string) (Config, error) {
	cfg := defaultConfig()
	var flags parseFlags
	fs := newFlagSet(&cfg, &flags)

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg.APITokens = splitCSV(flags.apiTokenCSV)
	cfg.ModelIDs = splitCSV(flags.modelCSV)
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = DefaultModel
	}
	if len(cfg.ModelIDs) == 0 {
		cfg.ModelIDs = []string{cfg.DefaultModel}
	} else if !contains(cfg.ModelIDs, cfg.DefaultModel) {
		cfg.ModelIDs = append([]string{cfg.DefaultModel}, cfg.ModelIDs...)
	}
	if cfg.DeepSeekBaseURL == "" {
		cfg.DeepSeekBaseURL = DefaultDeepSeekBaseURL
	}
	if flags.timeoutSeconds <= 0 {
		return Config{}, fmt.Errorf("--deepseek-http-timeout must be positive")
	}
	if cfg.DeepSeekMaxIdleConns < 0 {
		return Config{}, fmt.Errorf("--deepseek-max-idle-conns must be non-negative")
	}
	if cfg.DeepSeekMaxIdleConnsPerHost < 0 {
		return Config{}, fmt.Errorf("--deepseek-max-idle-conns-per-host must be non-negative")
	}
	if cfg.DeepSeekMaxConnsPerHost < 0 {
		return Config{}, fmt.Errorf("--deepseek-max-conns-per-host must be non-negative")
	}
	if cfg.DeepSeekMaxResponseBodyBytes < 0 {
		return Config{}, fmt.Errorf("--deepseek-max-response-body-bytes must be non-negative")
	}
	if cfg.StoreMaxResponses < 0 {
		return Config{}, fmt.Errorf("--store-max-responses must be non-negative")
	}
	if cfg.StoreMaxChatCompletions < 0 {
		return Config{}, fmt.Errorf("--store-max-chat-completions must be non-negative")
	}
	if cfg.StoreMaxConversations < 0 {
		return Config{}, fmt.Errorf("--store-max-conversations must be non-negative")
	}
	if flags.storeTTLSeconds < 0 {
		return Config{}, fmt.Errorf("--store-ttl must be non-negative")
	}
	if flags.storePruneIntervalSeconds < 0 {
		return Config{}, fmt.Errorf("--store-prune-interval must be non-negative")
	}
	if cfg.MaxRequestBodyBytes < 0 {
		return Config{}, fmt.Errorf("--max-request-body-bytes must be non-negative")
	}
	if flags.readHeaderTimeoutSeconds <= 0 {
		return Config{}, fmt.Errorf("--read-header-timeout must be positive")
	}
	if flags.idleTimeoutSeconds <= 0 {
		return Config{}, fmt.Errorf("--idle-timeout must be positive")
	}
	cfg.DeepSeekHTTPTimeout = time.Duration(flags.timeoutSeconds * float64(time.Second))
	cfg.StoreTTL = time.Duration(flags.storeTTLSeconds * float64(time.Second))
	cfg.StorePruneInterval = time.Duration(flags.storePruneIntervalSeconds * float64(time.Second))
	cfg.ReadHeaderTimeout = time.Duration(flags.readHeaderTimeoutSeconds * float64(time.Second))
	cfg.IdleTimeout = time.Duration(flags.idleTimeoutSeconds * float64(time.Second))
	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		DeepSeekMaxIdleConns:         200,
		DeepSeekMaxIdleConnsPerHost:  100,
		DeepSeekMaxResponseBodyBytes: 32 << 20,
		StoreMaxResponses:            1000,
		StoreMaxChatCompletions:      1000,
		StoreMaxConversations:        1000,
		StoreTTL:                     time.Hour,
		StorePruneInterval:           time.Minute,
		MaxRequestBodyBytes:          16 << 20,
		VerifySSL:                    true,
	}
}

func newFlagSet(cfg *Config, flags *parseFlags) *flag.FlagSet {
	fs := flag.NewFlagSet("deepseek-compatible", flag.ContinueOnError)
	fs.StringVar(&cfg.Listen, "listen", ":8080", "HTTP listen address")
	fs.StringVar(&flags.apiTokenCSV, "api-token", "", "comma-separated local bearer token list")
	fs.StringVar(&cfg.DeepSeekAPIKey, "deepseek-api-key", "", "DeepSeek upstream API key")
	fs.StringVar(&cfg.DeepSeekBaseURL, "deepseek-base-url", DefaultDeepSeekBaseURL, "DeepSeek upstream base URL")
	fs.StringVar(&cfg.DefaultModel, "deepseek-model", DefaultModel, "default DeepSeek model")
	fs.StringVar(&flags.modelCSV, "deepseek-models", "", "comma-separated model IDs exposed by /v1/models")
	fs.Float64Var(&flags.timeoutSeconds, "deepseek-http-timeout", 120, "DeepSeek HTTP timeout in seconds")
	fs.IntVar(&cfg.DeepSeekMaxIdleConns, "deepseek-max-idle-conns", cfg.DeepSeekMaxIdleConns, "maximum idle upstream HTTP connections")
	fs.IntVar(&cfg.DeepSeekMaxIdleConnsPerHost, "deepseek-max-idle-conns-per-host", cfg.DeepSeekMaxIdleConnsPerHost, "maximum idle upstream HTTP connections per host")
	fs.IntVar(&cfg.DeepSeekMaxConnsPerHost, "deepseek-max-conns-per-host", 0, "maximum upstream HTTP connections per host; 0 means unlimited")
	fs.Int64Var(&cfg.DeepSeekMaxResponseBodyBytes, "deepseek-max-response-body-bytes", cfg.DeepSeekMaxResponseBodyBytes, "maximum DeepSeek upstream response body size in bytes; 0 means unlimited")
	fs.IntVar(&cfg.StoreMaxResponses, "store-max-responses", cfg.StoreMaxResponses, "maximum locally stored Responses; 0 means unlimited")
	fs.IntVar(&cfg.StoreMaxChatCompletions, "store-max-chat-completions", cfg.StoreMaxChatCompletions, "maximum locally stored Chat Completions; 0 means unlimited")
	fs.IntVar(&cfg.StoreMaxConversations, "store-max-conversations", cfg.StoreMaxConversations, "maximum locally stored Conversations; 0 means unlimited")
	fs.Float64Var(&flags.storeTTLSeconds, "store-ttl", cfg.StoreTTL.Seconds(), "local store TTL in seconds after last access; 0 disables TTL")
	fs.Float64Var(&flags.storePruneIntervalSeconds, "store-prune-interval", cfg.StorePruneInterval.Seconds(), "minimum interval between request-path store prune checks in seconds; 0 disables request-path pruning")
	fs.Int64Var(&cfg.MaxRequestBodyBytes, "max-request-body-bytes", cfg.MaxRequestBodyBytes, "maximum local request body size in bytes; 0 means unlimited")
	fs.Float64Var(&flags.readHeaderTimeoutSeconds, "read-header-timeout", 10, "local HTTP read header timeout in seconds")
	fs.Float64Var(&flags.idleTimeoutSeconds, "idle-timeout", 120, "local HTTP idle timeout in seconds")
	fs.BoolVar(&cfg.DebugPprof, "debug-pprof", false, "enable authenticated /debug/pprof/ and /debug/vars endpoints")
	fs.BoolVar(&cfg.DebugLogBody, "debug-log-body", false, "log redacted request/response bodies")
	fs.BoolVar(&cfg.VerifySSL, "verify-ssl", true, "verify DeepSeek upstream TLS certificates")
	fs.Usage = func() { usage(fs) }
	return fs
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func usage(fs *flag.FlagSet) {
	out := fs.Output()
	fmt.Fprintf(out, "Usage:\n")
	fmt.Fprintf(out, "  %s [flags]\n\n", fs.Name())
	fmt.Fprintf(out, "Example:\n")
	fmt.Fprintf(out, "  %s --listen :8080 --api-token sk-local-test --deepseek-api-key sk-your-deepseek-key\n\n", fs.Name())
	fmt.Fprintf(out, "Flags:\n")
	printFlagDefaults(fs)
	fmt.Fprintf(out, "\nContainer deployment:\n")
	fmt.Fprintf(out, "  docker-entrypoint.sh maps environment variables to the same flags. See docker.env.example.\n\n")
	fmt.Fprintf(out, "Compatible APIs:\n")
	fmt.Fprintf(out, "  DeepSeek Chat Completions: POST /chat/completions\n")
	fmt.Fprintf(out, "  OpenAI Chat Completions:   /v1/chat/completions\n")
	fmt.Fprintf(out, "  OpenAI Responses:          /v1/responses\n")
	fmt.Fprintf(out, "  OpenAI Conversations:      /v1/conversations\n")
	fmt.Fprintf(out, "  Anthropic Messages:        /v1/messages\n")
	fmt.Fprintf(out, "  Gemini Generate Content:   /v1beta/models/{model}:generateContent, /v1/models/{model}:generateContent\n")
	fmt.Fprintf(out, "  Common endpoints:          /v1/models, /health, /health/memory\n")
}

func printFlagDefaults(fs *flag.FlagSet) {
	out := fs.Output()
	fs.VisitAll(func(f *flag.Flag) {
		name, usage := flag.UnquoteUsage(f)
		if name == "" {
			fmt.Fprintf(out, "  --%s\n", f.Name)
		} else {
			fmt.Fprintf(out, "  --%s %s\n", f.Name, name)
		}
		if usage != "" && f.DefValue != "" {
			fmt.Fprintf(out, "      %s (default %s)\n", usage, f.DefValue)
		} else if usage != "" {
			fmt.Fprintf(out, "      %s\n", usage)
		} else if f.DefValue != "" {
			fmt.Fprintf(out, "      default %s\n", f.DefValue)
		}
	})
}
