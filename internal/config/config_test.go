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
	"reflect"
	"testing"
	"time"
)

func TestParseDefaults(t *testing.T) {
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
		"--read-header-timeout", "3.5",
		"--idle-timeout", "45",
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
	if cfg.ReadHeaderTimeout != 3500*time.Millisecond {
		t.Fatalf("ReadHeaderTimeout = %s", cfg.ReadHeaderTimeout)
	}
	if cfg.IdleTimeout != 45*time.Second {
		t.Fatalf("IdleTimeout = %s", cfg.IdleTimeout)
	}
	if !cfg.DebugLogBody {
		t.Fatalf("boolean flags were not parsed: %#v", cfg)
	}
	if cfg.VerifySSL {
		t.Fatalf("VerifySSL = %t", cfg.VerifySSL)
	}
}

func TestParsePrependsDefaultModelWhenMissingFromModelList(t *testing.T) {
	cfg, err := Parse([]string{"--deepseek-model", "deepseek-main", "--deepseek-models", "deepseek-alt"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.ModelIDs, []string{"deepseek-main", "deepseek-alt"}) {
		t.Fatalf("ModelIDs = %#v", cfg.ModelIDs)
	}
}

func TestParseRejectsNonPositiveTimeout(t *testing.T) {
	if _, err := Parse([]string{"--deepseek-http-timeout", "0"}); err == nil {
		t.Fatal("expected error for zero timeout")
	}
}

func TestParseRejectsInvalidConnectionLimits(t *testing.T) {
	for _, flag := range []string{"--deepseek-max-idle-conns", "--deepseek-max-idle-conns-per-host", "--deepseek-max-conns-per-host"} {
		if _, err := Parse([]string{flag, "-1"}); err == nil {
			t.Fatalf("expected error for %s", flag)
		}
	}
}

func TestParseRejectsNonPositiveServerTimeouts(t *testing.T) {
	for _, flag := range []string{"--read-header-timeout", "--idle-timeout"} {
		if _, err := Parse([]string{flag, "0"}); err == nil {
			t.Fatalf("expected error for %s", flag)
		}
	}
}
