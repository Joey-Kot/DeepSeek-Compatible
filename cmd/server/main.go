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

package main

import (
	"log"
	"net/http"
	"os"

	"deepseek-responses-compatible/internal/config"
	"deepseek-responses-compatible/internal/httpapi"
	"deepseek-responses-compatible/internal/state"
	"deepseek-responses-compatible/internal/upstream/deepseek"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	upstream := deepseek.New(cfg.DeepSeekBaseURL, cfg.DeepSeekAPIKey, cfg.DeepSeekHTTPTimeout, cfg.VerifySSL)
	upstream.DebugLogBody = cfg.DebugLogBody
	server := httpapi.New(cfg, upstream, state.New())
	log.Printf("listening on %s", cfg.Listen)
	if err := http.ListenAndServe(cfg.Listen, server); err != nil {
		log.Fatal(err)
	}
}
