package parsers

import (
	"strings"
)

/*
KV parsers focus on parsing key=val type logic with various separators
A lot of these IOT devices use some psuedo-made-up formats that need these specialized parsers.
For anything JSON, use `gjson`
*/

// Parse `k=v`
func ParseOneKV(text string) (key, val string) {
	text = strings.TrimSpace(text)
	idx := strings.IndexByte(text, '=')
	if idx < 0 {
		return text, ""
	}
	return text[:idx], text[idx+1:]
}

// Parse many `k=v` with a `delim` (eg newline)
func ParseManyKV(text, delim string) (ret map[string]string) {
	ret = make(map[string]string)
	for _, line := range strings.Split(text, delim) {
		line = strings.TrimSpace(line)
		if line != "" {
			k, v := ParseOneKV(line)
			ret[k] = v
		}
	}
	return
}
