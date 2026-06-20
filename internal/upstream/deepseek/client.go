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
	BaseURL              string
	APIKey               string
	Timeout              time.Duration
	MaxResponseBodyBytes int64
	DebugLogBody         bool
	HTTPClient           *http.Client
}

type TransportConfig struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
}

func DefaultTransportConfig() TransportConfig {
	return TransportConfig{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     0,
	}
}

func New(baseURL, apiKey string, timeout time.Duration, verifySSL bool) *Client {
	return NewWithTransportConfig(baseURL, apiKey, timeout, verifySSL, DefaultTransportConfig())
}

func NewWithTransportConfig(baseURL, apiKey string, timeout time.Duration, verifySSL bool, transportConfig TransportConfig) *Client {
	return &Client{
		BaseURL:              baseURL,
		APIKey:               apiKey,
		Timeout:              timeout,
		MaxResponseBodyBytes: 32 << 20,
		HTTPClient:           newHTTPClient(timeout, verifySSL, transportConfig),
	}
}

func (c *Client) CloseIdleConnections() {
	if c.HTTPClient != nil {
		c.HTTPClient.CloseIdleConnections()
	}
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
	body, err := c.readResponseBody(resp.Body)
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
		body, err := c.readResponseBody(resp.Body)
		if err != nil {
			return err
		}
		if c.DebugLogBody {
			log.Printf("debug body deepseek stream error status=%d body=%s", resp.StatusCode, debuglog.Body(body))
		}
		var data shared.Map
		_ = json.Unmarshal(body, &data)
		return HTTPError{StatusCode: resp.StatusCode, Message: deepseekErrorMessage(data, string(body))}
	}

	scanner := bufio.NewScanner(c.responseBodyReader(resp.Body))
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

func (c *Client) readResponseBody(reader io.Reader) ([]byte, error) {
	if c.MaxResponseBodyBytes <= 0 {
		return io.ReadAll(reader)
	}
	limited := io.LimitReader(reader, c.MaxResponseBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > c.MaxResponseBodyBytes {
		return nil, responseTooLargeError()
	}
	return body, nil
}

func (c *Client) responseBodyReader(reader io.Reader) io.Reader {
	if c.MaxResponseBodyBytes <= 0 {
		return reader
	}
	return &maxBytesReader{reader: reader, max: c.MaxResponseBodyBytes}
}

type maxBytesReader struct {
	reader io.Reader
	max    int64
	read   int64
}

func (r *maxBytesReader) Read(p []byte) (int, error) {
	remaining := r.max - r.read
	if remaining < 0 {
		return 0, responseTooLargeError()
	}
	limit := remaining + 1
	if int64(len(p)) > limit {
		p = p[:limit]
	}
	n, err := r.reader.Read(p)
	if int64(n) > remaining {
		r.read = r.max + 1
		return 0, responseTooLargeError()
	}
	r.read += int64(n)
	return n, err
}

func responseTooLargeError() error {
	return HTTPError{StatusCode: http.StatusBadGateway, Message: "DeepSeek response body is too large"}
}

func newHTTPClient(timeout time.Duration, verifySSL bool, config TransportConfig) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if config.MaxIdleConns > 0 {
		transport.MaxIdleConns = config.MaxIdleConns
	}
	if config.MaxIdleConnsPerHost > 0 {
		transport.MaxIdleConnsPerHost = config.MaxIdleConnsPerHost
	}
	if config.MaxConnsPerHost > 0 {
		transport.MaxConnsPerHost = config.MaxConnsPerHost
	}
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
