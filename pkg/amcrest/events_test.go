package amcrest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePayload(t *testing.T) {
	parts := parseStreamPayload([]byte("a=b;c=d;eq=this\nis\ntest"))
	assert.Len(t, parts, 3)
	assert.Equal(t, map[string]string{
		"a":  "b",
		"c":  "d",
		"eq": "this\nis\ntest",
	}, parts)

	parts = parseStreamPayload([]byte("a=b"))
	assert.Len(t, parts, 1)
	assert.Equal(t, map[string]string{
		"a": "b",
	}, parts)

	parts = parseStreamPayload([]byte(""))
	assert.Len(t, parts, 0)
	assert.Equal(t, map[string]string{}, parts)
}
