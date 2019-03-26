package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringSet(t *testing.T) {
	assert := assert.New(t)

	indexed := StringSet([]string{"abc", "def"})

	assert.Len(indexed, 2)
	assert.Equal(indexed["abc"], "abc")
	assert.Equal(indexed["def"], "def")
}
