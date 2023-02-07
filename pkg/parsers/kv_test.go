package parsers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindUnquotedChar(t *testing.T) {
	assert.Equal(t, -1, findNextUnquotedChar(`abcd="qef"`, ','))
	assert.Equal(t, 10, findNextUnquotedChar(`abcd="qef",dog=wag`, ','))
	assert.Equal(t, 18, findNextUnquotedChar(`abcd="qef,inquote",dog=wag`, ','))
}

func TestParseKV(t *testing.T) {
	k, v := ParseOneKV("a=b")
	assert.Equal(t, "a", k)
	assert.Equal(t, "b", v)

	k, v = ParseOneKV("abc")
	assert.Equal(t, "abc", k)
	assert.Equal(t, "", v)

	k, v = ParseOneKV(`abc=123`)
	assert.Equal(t, "abc", k)
	assert.Equal(t, "123", v)

	k, v = ParseOneKV(`abc="123"`)
	assert.Equal(t, "abc", k)
	assert.Equal(t, "123", v)

	k, v = ParseOneKV(`abc=123=qef`)
	assert.Equal(t, "abc", k)
	assert.Equal(t, "123=qef", v)
}

func TestParseManyKVs(t *testing.T) {
	assert.Equal(t, map[string]string{}, ParseManyKV("", ','))
	assert.Equal(t, map[string]string{"a": "b"}, ParseManyKV("a=b", ','))
	assert.Equal(t, map[string]string{"a": "b"}, ParseManyKV("a=\"b\"", ','))
	assert.Equal(t, map[string]string{"a": "b,c", "d": "ef", "g": "hi"}, ParseManyKV(`a="b,c",d=ef,g=hi`, ','))
}
