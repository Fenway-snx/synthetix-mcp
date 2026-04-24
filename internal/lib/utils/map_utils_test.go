package utils

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetStringFromMap(t *testing.T) {
	testMap := map[string]any{
		"validKey":     "value2",
		"emptyKey":     "",
		"wrongTypeKey": 5,
	}

	value, provided, err := GetStringFromMap(testMap, "validKey")
	assert.Equal(t, "value2", value)
	assert.Equal(t, "value2", provided)
	assert.Nil(t, err)

	value, provided, err = GetStringFromMap(testMap, "emptyKey")
	assert.Equal(t, value, "")
	assert.Equal(t, "", provided)
	assert.Error(t, err)

	value, provided, err = GetStringFromMap(testMap, "wrongTypeKey")
	assert.Equal(t, value, "")
	assert.Equal(t, 5, provided)
	assert.Error(t, err)

	value, provided, err = GetStringFromMap(testMap, "nonexistantKey")
	assert.Equal(t, value, "")
	assert.Nil(t, provided)
	assert.Error(t, err)
}

func Test_MapKeys(t *testing.T) {
	testMap := map[string]string{
		"key2": "value2",
		"key1": "value1",
		"key3": "value3",
	}

	keys := MapKeys(testMap)
	sort.Strings(keys)
	assert.Equal(t, []string{"key1", "key2", "key3"}, keys)
}

func Test_MapValues(t *testing.T) {
	testMap := map[string]string{
		"key2": "value2",
		"key1": "value1",
		"key3": "value3",
	}

	values := MapValues(testMap)
	sort.Strings(values)
	assert.Equal(t, []string{"value1", "value2", "value3"}, values)
}
