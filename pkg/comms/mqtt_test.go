package comms

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathCombineAssumption(t *testing.T) {
	assert.Equal(t, "abc/efg", path.Join("abc", "efg"))
	assert.Equal(t, "abc/efg", path.Join("abc", "/efg"))
	assert.Equal(t, "abc/efg", path.Join("abc/", "/efg"))
}
