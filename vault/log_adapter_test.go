package vault

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogAdapterFlatten(t *testing.T) {
	assert.Equal(t, flatten([]map[string]interface{}{}), []interface{}{}, "empty array")

	assert.Equal(t, flatten([]map[string]interface{}{{"key1": "val"}, {}, {"key1": "val2"}}), []interface{}{"key1", "val", "key1", "val2"})
}
