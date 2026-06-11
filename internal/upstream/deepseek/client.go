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

package deepseek

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"deepseek-responses-compatible/internal/adapters/openai/shared"
	"deepseek-responses-compatible/internal/debuglog"
)

type Client struct {
	BaseURL      string
	APIKey       string
	Timeout      time.Duration
	DebugLogBody bool
	HTTPClient   *http.Client
}

func New(baseURL, apiKey string, timeout time.Duration, verifySSL bool) *Client {
	return &Client{BaseURL: baseURL, APIKey: apiKey, Timeout: timeout, HTTPClient: newHTTPClient(timeout, verifySSL)}
}

func (c *Client) Chat(ctx context.Context, payload shared.Map) (shared.Map, error) {
	req, err := c.newRequest(ctx, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.DebugLogBody {
		log.Printf("debug body deepseek request url=%s body=%s", req.URL.String(), debuglog.MarshalBody(payload))
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("external DeepSeek request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if c.DebugLogBody {
		log.Printf("debug body deepseek response status=%d body=%s", resp.StatusCode, debuglog.Body(body))
	}
	var data shared.Map
	_ = json.Unmarshal(body, &data)
	if resp.StatusCode >= 400 {
		return nil, HTTPError{StatusCode: resp.StatusCode, Message: deepseekErrorMessage(data, string(body))}
	}
	if data == nil {
		return nil, HTTPError{StatusCode: http.StatusBadGateway, Message: "DeepSeek returned a non-JSON response"}
	}
	return data, nil
}

func (c *Client) StreamChat(ctx context.Context, payload shared.Map, handle func(shared.Map) error) error {
	req, err := c.newRequest(ctx, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.DebugLogBody {
		log.Printf("debug body deepseek stream request url=%s body=%s", req.URL.String(), debuglog.MarshalBody(payload))
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("external DeepSeek stream failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		if c.DebugLogBody {
			log.Printf("debug body deepseek stream error status=%d body=%s", resp.StatusCode, debuglog.Body(body))
		}
		var data shared.Map
		_ = json.Unmarshal(body, &data)
		return HTTPError{StatusCode: resp.StatusCode, Message: deepseekErrorMessage(data, string(body))}
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		text := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if c.DebugLogBody {
			log.Printf("debug body deepseek stream response data=%s", debuglog.Body([]byte(text)))
		}
		if text == "[DONE]" {
			break
		}
		var chunk shared.Map
		if err := json.Unmarshal([]byte(text), &chunk); err != nil {
			continue
		}
		if err := handle(chunk); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (c *Client) newRequest(ctx context.Context, payload shared.Map) (*http.Request, error) {
	if c.APIKey == "" {
		return nil, HTTPError{StatusCode: http.StatusInternalServerError, Message: "DEEPSEEK API key is not configured"}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.chatURL(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *Client) chatURL() string {
	base := strings.TrimRight(c.BaseURL, "/")
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	return base + "/chat/completions"
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: c.Timeout}
}

func newHTTPClient(timeout time.Duration, verifySSL bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if !verifySSL {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e HTTPError) Error() string {
	return e.Message
}

func deepseekErrorMessage(data shared.Map, fallback string) string {
	if data != nil {
		if errObj, ok := data["error"].(map[string]any); ok {
			if message := shared.StringValue(errObj["message"]); message != "" {
				return "External DeepSeek error: " + message
			}
		}
		if message := shared.StringValue(data["message"]); message != "" {
			return "External DeepSeek error: " + message
		}
		if code := shared.StringValue(data["code"]); code != "" {
			return "External DeepSeek error: " + code
		}
	}
	return "External DeepSeek error: " + fallback
}
