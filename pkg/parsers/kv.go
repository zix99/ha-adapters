package parsers

import (
	"strings"
)

/*
KV parsers focus on parsing key=val type logic with various separators
A lot of these IOT devices use some psuedo-made-up formats that need these specialized parsers.
For anything JSON, use `gjson`
*/

// Parse `k=v`, trimming any nonsense (spaces)
func ParseOneKV(text string) (key, val string) {
	text = strings.TrimSpace(text)
	idx := strings.IndexByte(text, '=')
	if idx < 0 {
		return text, ""
	}
	return text[:idx], strings.Trim(text[idx+1:], "\"")
}

// Parse many `k=v` with a `delim` (eg newline)
func ParseManyKV(text string, delim byte) (ret map[string]string) {
	ret = make(map[string]string)

	for {
		next := findNextUnquotedChar(text, delim)
		if next < 0 {
			break
		}

		line := text[:next]
		if line != "" {
			k, v := ParseOneKV(line)
			ret[k] = v
		}

		text = text[next+1:]
	}

	if len(text) > 0 {
		k, v := ParseOneKV(text)
		ret[k] = v
	}
	return
}

func findNextUnquotedChar(s string, find byte) int {
	quoted := false
	for i, c := range s {
		if c == '"' {
			quoted = !quoted
		}
		if !quoted && byte(c) == find {
			return i
		}
	}
	return -1
}
