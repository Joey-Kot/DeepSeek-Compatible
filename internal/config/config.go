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
	Listen              string
	APITokens           []string
	DeepSeekAPIKey      string
	DeepSeekBaseURL     string
	DefaultModel        string
	ModelIDs            []string
	DeepSeekHTTPTimeout time.Duration
	DebugLogBody        bool
	VerifySSL           bool
}

func Parse(args []string) (Config, error) {
	fs := flag.NewFlagSet("deepseek-responses-compatible", flag.ContinueOnError)

	var apiTokenCSV string
	var modelCSV string
	var timeoutSeconds float64
	cfg := Config{VerifySSL: true}

	fs.StringVar(&cfg.Listen, "listen", ":8080", "HTTP listen address")
	fs.StringVar(&apiTokenCSV, "api-token", "", "comma-separated local bearer token list")
	fs.StringVar(&cfg.DeepSeekAPIKey, "deepseek-api-key", "", "DeepSeek upstream API key")
	fs.StringVar(&cfg.DeepSeekBaseURL, "deepseek-base-url", DefaultDeepSeekBaseURL, "DeepSeek upstream base URL")
	fs.StringVar(&cfg.DefaultModel, "deepseek-model", DefaultModel, "default DeepSeek model")
	fs.StringVar(&modelCSV, "deepseek-models", "", "comma-separated model IDs exposed by /v1/models")
	fs.Float64Var(&timeoutSeconds, "deepseek-http-timeout", 120, "DeepSeek HTTP timeout in seconds")
	fs.BoolVar(&cfg.DebugLogBody, "debug-log-body", false, "log redacted request/response bodies")
	fs.BoolVar(&cfg.VerifySSL, "verify-ssl", true, "verify DeepSeek upstream TLS certificates")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg.APITokens = splitCSV(apiTokenCSV)
	cfg.ModelIDs = splitCSV(modelCSV)
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
	if timeoutSeconds <= 0 {
		return Config{}, fmt.Errorf("--deepseek-http-timeout must be positive")
	}
	cfg.DeepSeekHTTPTimeout = time.Duration(timeoutSeconds * float64(time.Second))
	return cfg, nil
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
