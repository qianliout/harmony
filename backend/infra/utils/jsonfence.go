package utils

import (
	"encoding/json"
	"strings"

	"harmony/infra/consts"
)

// StripLLMJSONFence 在 json.Unmarshal 前剥掉 LLM 回复外层的代码围栏与前后说明文字。
func StripLLMJSONFence(s string) string {
	raw := stripLLMOuterFence(s)
	raw = extractJSONPayload(raw)
	return raw
}

// UnmarshalLLMJSON 剥掉围栏后反序列化到 dest；剥完为空时按空对象处理。
func UnmarshalLLMJSON(content string, dest any) error {
	raw := StripLLMJSONFence(content)
	if raw == "" {
		raw = "{}"
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return err
	}
	return nil
}

func stripLLMOuterFence(s string) string {
	raw := strings.TrimSpace(s)
	body, ok := fencedBody(raw)
	if ok {
		return body
	}

	raw = strings.TrimPrefix(raw, consts.LLMFence+"json")
	raw = strings.TrimPrefix(raw, consts.LLMFence+"JSON")
	raw = strings.TrimPrefix(raw, consts.LLMFence+"Json")
	raw = strings.TrimPrefix(raw, consts.LLMFence)
	raw = strings.TrimSuffix(raw, consts.LLMFence)
	raw = strings.TrimSpace(raw)
	return raw
}

// fencedBody 取第一对完整围栏之间的内容并去掉开头的语言标记；围栏不成对时 ok 为 false。
func fencedBody(raw string) (string, bool) {
	start := strings.Index(raw, consts.LLMFence)
	if start < 0 {
		return "", false
	}
	after := raw[start+len(consts.LLMFence):]
	after = strings.TrimLeft(after, "\r\n")
	langLen := len(consts.LLMFenceLang)
	hasLang := len(after) >= langLen && strings.EqualFold(after[:langLen], consts.LLMFenceLang)
	if hasLang {
		after = after[langLen:]
		after = strings.TrimLeft(after, "\r\n\t ")
	}
	end := strings.Index(after, consts.LLMFence)
	if end < 0 {
		return "", false
	}
	body := strings.TrimSpace(after[:end])
	return body, true
}

// extractJSONPayload 从第一个 { 或 [ 起截到与之配平的括号为止，跳过字符串字面量里的括号；配不平时原样返回。
func extractJSONPayload(s string) string {
	if s == "" {
		return s
	}

	braceIdx := strings.IndexByte(s, '{')
	bracketIdx := strings.IndexByte(s, '[')
	startIdx := -1
	var openCh byte
	var closeCh byte
	switch {
	case braceIdx < 0 && bracketIdx < 0:
		return s
	case braceIdx < 0:
		startIdx, openCh, closeCh = bracketIdx, '[', ']'
	case bracketIdx < 0:
		startIdx, openCh, closeCh = braceIdx, '{', '}'
	case braceIdx < bracketIdx:
		startIdx, openCh, closeCh = braceIdx, '{', '}'
	default:
		startIdx, openCh, closeCh = bracketIdx, '[', ']'
	}

	body := s[startIdx:]
	depth := 0
	inString := false
	escape := false
	for i := range len(body) {
		ch := body[i]
		if escape {
			escape = false
			continue
		}
		if ch == '\\' && inString {
			escape = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == openCh {
			depth++
			continue
		}
		if ch != closeCh {
			continue
		}
		depth--
		if depth == 0 {
			body = strings.TrimSpace(body[:i+1])
			return body
		}
	}
	return s
}

func CompactStringList(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]string)
	for i := range items {
		item := strings.TrimSpace(items[i])
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = item
		out = append(out, item)
	}
	return out
}

func MarshalStringList(items []string) string {
	items = CompactStringList(items)
	if len(items) == 0 {
		return consts.EmptyJSONArray
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return consts.EmptyJSONArray
	}
	return string(payload)
}

func UnmarshalStringList(text string) []string {
	out := make([]string, 0)
	text = strings.TrimSpace(text)
	if text == "" {
		return out
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return make([]string, 0)
	}
	compacted := CompactStringList(out)
	return compacted
}
