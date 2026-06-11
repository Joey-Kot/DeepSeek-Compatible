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

package deepseekv3

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"sync"
)

//go:embed tokenizer.json
var tokenizerJSON []byte

type tokenizerFile struct {
	AddedTokens []addedToken `json:"added_tokens"`
	Model       modelConfig  `json:"model"`
}

type addedToken struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

type modelConfig struct {
	Vocab  map[string]int `json:"vocab"`
	Merges []string       `json:"merges"`
}

type Tokenizer struct {
	vocab         map[string]int
	ranks         map[string]int
	added         map[string]int
	addedByPrefix map[byte][]string
	byteEncoder   [256]string
	digitPattern  *regexp.Regexp
	cjkPattern    *regexp.Regexp
	textPattern   *regexp.Regexp
}

type Message struct {
	Role      string
	Content   string
	ToolCalls []ToolCall
}

type ToolCall struct {
	Type      string
	Name      string
	Arguments string
}

var (
	defaultOnce      sync.Once
	defaultTokenizer *Tokenizer
	defaultErr       error
)

func Default() (*Tokenizer, error) {
	defaultOnce.Do(func() {
		defaultTokenizer, defaultErr = New(tokenizerJSON)
	})
	return defaultTokenizer, defaultErr
}

func New(data []byte) (*Tokenizer, error) {
	var parsed tokenizerFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	t := &Tokenizer{
		vocab:         parsed.Model.Vocab,
		ranks:         map[string]int{},
		added:         map[string]int{},
		addedByPrefix: map[byte][]string{},
		digitPattern:  regexp.MustCompile(`\p{N}{1,3}`),
		cjkPattern:    regexp.MustCompile(`[一-龥぀-ゟ゠-ヿ]+`),
		textPattern:   regexp.MustCompile("[!\"#$%&'()*+,\\-./:;<=>?@\\[\\\\\\]^_`{|}~][A-Za-z]+|[^\r\n\\p{L}\\p{P}\\p{S}]?[\\p{L}\\p{M}]+| ?[\\p{P}\\p{S}]+[\r\n]*|\\s*[\r\n]+|\\s+"),
	}
	for rank, merge := range parsed.Model.Merges {
		parts := strings.Split(merge, " ")
		if len(parts) == 2 {
			t.ranks[pairKey(parts[0], parts[1])] = rank
		}
	}
	for _, tok := range parsed.AddedTokens {
		if tok.Content == "" {
			continue
		}
		t.added[tok.Content] = tok.ID
		t.addedByPrefix[tok.Content[0]] = append(t.addedByPrefix[tok.Content[0]], tok.Content)
	}
	for prefix := range t.addedByPrefix {
		sort.Slice(t.addedByPrefix[prefix], func(i, j int) bool {
			return len(t.addedByPrefix[prefix][i]) > len(t.addedByPrefix[prefix][j])
		})
	}
	t.byteEncoder = byteEncoder()
	return t, nil
}

func CountText(text string) int {
	t, err := Default()
	if err != nil {
		return 0
	}
	return t.Count(text)
}

func CountMessages(messages []Message) int {
	t, err := Default()
	if err != nil {
		return 0
	}
	return t.Count(RenderChat(messages, true))
}

func (t *Tokenizer) Count(text string) int {
	if text == "" {
		return 0
	}
	count := 0
	for _, segment := range t.splitAdded(text) {
		if _, ok := t.added[segment]; ok {
			count++
			continue
		}
		for _, token := range t.preTokenize(segment) {
			if token == "" {
				continue
			}
			count += t.countPretoken(token)
		}
	}
	return count
}

func (t *Tokenizer) splitAdded(text string) []string {
	out := []string{}
	for len(text) > 0 {
		if candidates := t.addedByPrefix[text[0]]; len(candidates) > 0 {
			matched := ""
			for _, candidate := range candidates {
				if strings.HasPrefix(text, candidate) {
					matched = candidate
					break
				}
			}
			if matched != "" {
				out = append(out, matched)
				text = text[len(matched):]
				continue
			}
		}
		next := len(text)
		for prefix := range t.addedByPrefix {
			if idx := strings.IndexByte(text[1:], prefix); idx >= 0 && idx+1 < next {
				next = idx + 1
			}
		}
		out = append(out, text[:next])
		text = text[next:]
	}
	return out
}

func (t *Tokenizer) preTokenize(text string) []string {
	parts := []string{text}
	for _, pattern := range []*regexp.Regexp{t.digitPattern, t.cjkPattern, t.textPattern} {
		parts = splitIsolated(parts, pattern)
	}
	return parts
}

func splitIsolated(parts []string, pattern *regexp.Regexp) []string {
	out := []string{}
	for _, part := range parts {
		matches := pattern.FindAllStringIndex(part, -1)
		if len(matches) == 0 {
			out = append(out, part)
			continue
		}
		offset := 0
		for _, match := range matches {
			if match[0] > offset {
				out = append(out, part[offset:match[0]])
			}
			out = append(out, part[match[0]:match[1]])
			offset = match[1]
		}
		if offset < len(part) {
			out = append(out, part[offset:])
		}
	}
	return out
}

func (t *Tokenizer) countPretoken(token string) int {
	encoded := byteLevelEncode(token, t.byteEncoder)
	if encoded == "" {
		return 0
	}
	if _, ok := t.vocab[encoded]; ok {
		return 1
	}
	pieces := t.bpe(encoded)
	if len(pieces) == 0 {
		return 0
	}
	return len(pieces)
}

func (t *Tokenizer) bpe(token string) []string {
	word := runePieces(token)
	if len(word) < 2 {
		return word
	}
	for {
		bestIndex := -1
		bestRank := int(^uint(0) >> 1)
		for i := 0; i < len(word)-1; i++ {
			if rank, ok := t.ranks[pairKey(word[i], word[i+1])]; ok && rank < bestRank {
				bestRank = rank
				bestIndex = i
			}
		}
		if bestIndex < 0 {
			break
		}
		merged := append([]string{}, word[:bestIndex]...)
		merged = append(merged, word[bestIndex]+word[bestIndex+1])
		merged = append(merged, word[bestIndex+2:]...)
		word = merged
		if len(word) < 2 {
			break
		}
	}
	return word
}

func runePieces(value string) []string {
	out := []string{}
	for _, r := range value {
		out = append(out, string(r))
	}
	return out
}

func byteLevelEncode(value string, encoder [256]string) string {
	var b strings.Builder
	for _, by := range []byte(value) {
		b.WriteString(encoder[by])
	}
	return b.String()
}

func byteEncoder() [256]string {
	var out [256]string
	used := map[int]bool{}
	values := []int{}
	for i := int('!'); i <= int('~'); i++ {
		values = append(values, i)
		used[i] = true
	}
	for i := 0xA1; i <= 0xAC; i++ {
		values = append(values, i)
		used[i] = true
	}
	for i := 0xAE; i <= 0xFF; i++ {
		values = append(values, i)
		used[i] = true
	}
	codes := append([]int{}, values...)
	n := 0
	for i := 0; i < 256; i++ {
		if !used[i] {
			values = append(values, i)
			codes = append(codes, 256+n)
			n++
		}
	}
	for i, value := range values {
		out[value] = string(rune(codes[i]))
	}
	return out
}

func pairKey(a, b string) string {
	return a + "\x00" + b
}
