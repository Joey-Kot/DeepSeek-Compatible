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
	"bytes"
	"expvar"
	"net/http"
	"net/http/pprof"
	"strings"

	"deepseek-responses-compatible/internal/debuglog"
)

func (s *Server) handleDebug(w http.ResponseWriter, r *http.Request) bool {
	if !s.cfg.DebugPprof {
		return false
	}
	switch {
	case r.URL.Path == "/debug/vars":
		expvar.Handler().ServeHTTP(w, r)
		return true
	case r.URL.Path == "/debug/pprof/" || r.URL.Path == "/debug/pprof":
		pprof.Index(w, r)
		return true
	case strings.HasPrefix(r.URL.Path, "/debug/pprof/cmdline"):
		pprof.Cmdline(w, r)
		return true
	case strings.HasPrefix(r.URL.Path, "/debug/pprof/profile"):
		pprof.Profile(w, r)
		return true
	case strings.HasPrefix(r.URL.Path, "/debug/pprof/symbol"):
		pprof.Symbol(w, r)
		return true
	case strings.HasPrefix(r.URL.Path, "/debug/pprof/trace"):
		pprof.Trace(w, r)
		return true
	case strings.HasPrefix(r.URL.Path, "/debug/pprof/"):
		pprof.Index(w, r)
		return true
	default:
		return false
	}
}

type debugResponseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func newDebugResponseWriter(w http.ResponseWriter) *debugResponseWriter {
	return &debugResponseWriter{ResponseWriter: w}
}

func (w *debugResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *debugResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	remaining := debuglog.MaxBodyBytes + 1 - w.body.Len()
	if remaining > 0 {
		if len(data) > remaining {
			w.body.Write(data[:remaining])
		} else {
			w.body.Write(data)
		}
	}
	return w.ResponseWriter.Write(data)
}

func (w *debugResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *debugResponseWriter) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *debugResponseWriter) bodyBytes() []byte {
	return w.body.Bytes()
}
