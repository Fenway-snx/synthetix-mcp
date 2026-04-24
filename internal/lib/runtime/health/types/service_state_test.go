package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_StateManager_INT_FORMS(t *testing.T) {
	assert.Equal(t, 0, int(ServiceState_Unknown), "Invalid service state value.")
	assert.Equal(t, 1, int(ServiceState_Healthy), "Invalid service state value.")
	assert.Equal(t, 2, int(ServiceState_ShuttingDown), "Invalid service state value.")
	assert.Equal(t, 3, int(ServiceState_Starting), "Invalid service state value.")
	assert.Equal(t, 4, int(ServiceState_Unhealthy), "Invalid service state value.")
	assert.Equal(t, 5, int(ServiceState_Draining), "Invalid service state value.")
	assert.Equal(t, 6, int(ServiceState_Idle), "Invalid service state value.")
}

type General struct {
	ServiceState any `json:"service_state"`
}

type Proper struct {
	ServiceState ServiceState `json:"service_state"`
}

func Test_StateManager_Unmarshal_STRING_FORMS(t *testing.T) {

	tests := []struct {
		name             string
		data             string
		expectedResponse ServiceState
	}{
		{"Draining should be case insensitive", "DrAiNiNg", ServiceState_Draining},
		{"Healthy should be case insensitive", "hEaltHy", ServiceState_Healthy},
		{"Idle should be case insensitive", "iDlE", ServiceState_Idle},
		{"Shutting Down should be case insensitive", "ShuTtinG_DoWn", ServiceState_ShuttingDown},
		{"Starting should be case insensitive", "StArTInG", ServiceState_Starting},
		{"Unhealthy should be case insensitive", "UnHeAlThY", ServiceState_Unhealthy},
		{"Unknown should be case insensitive", "UnKnOwN", ServiceState_Unknown},
		{"Any random stuff should cast to unknown", "Random State", ServiceState_Unknown},
		{"Any random stuff should cast to unknown", "BRUH", ServiceState_Unknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			thingo := General{
				ServiceState: test.data,
			}

			bytes, err := json.Marshal(thingo)
			assert.NoError(t, err, "Something went wrong on marshal")

			var parsedData Proper
			err = json.Unmarshal(bytes, &parsedData)
			assert.NoError(t, err, "Something went wrong on unmarshal")

			assert.Equal(t, test.expectedResponse, parsedData.ServiceState)
		})

	}
}

func Test_StateManager_Unmarshal_INT_FORMS(t *testing.T) {
	tests := []struct {
		name             string
		data             int
		expectedResponse ServiceState
	}{
		{"1 should be equal to Healthy", 1, ServiceState_Healthy},
		{"2 should be equal to shutting down", 2, ServiceState_ShuttingDown},
		{"3 should be equal to starting", 3, ServiceState_Starting},
		{"4 should be equal to unhealthy", 4, ServiceState_Unhealthy},
		{"5 should be equal to draining", 5, ServiceState_Draining},
		{"6 should be equal to idle", 6, ServiceState_Idle},
		{"7 should be equal to unknown", 7, ServiceState_Unknown},
		{"Any random stuff should be equal to unknown", 1_000, ServiceState_Unknown},
		{"Any random stuff should be equal to unknown", -1, ServiceState_Unknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			thingo := General{
				ServiceState: test.data,
			}

			bytes, err := json.Marshal(thingo)
			assert.NoError(t, err, "Something went wrong on marshal")

			var parsedData Proper
			err = json.Unmarshal(bytes, &parsedData)
			assert.NoError(t, err, "Something went wrong on unmarshal")

			assert.Equal(t, test.expectedResponse, parsedData.ServiceState)
		})

	}
}

func Test_StateManager_Unmarshal_INVALID_FORMS(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{"Bool should cause an error", false},
		{"Slice should cause an error", []string{}},
		{"Nil should cause an error", nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			thingo := General{
				ServiceState: test.data,
			}

			bytes, err := json.Marshal(thingo)
			assert.NoError(t, err, "Something went wrong on marshal")

			var parsedData Proper
			err = json.Unmarshal(bytes, &parsedData)
			assert.ErrorContains(t, err, "invalid type")
		})
	}
}

func Test_IsHaltOwnedState(t *testing.T) {
	assert.True(t, IsHaltOwnedState(ServiceState_Draining))
	assert.True(t, IsHaltOwnedState(ServiceState_Idle))
	assert.True(t, IsHaltOwnedState(ServiceState_ShuttingDown))

	assert.False(t, IsHaltOwnedState(ServiceState_Healthy))
	assert.False(t, IsHaltOwnedState(ServiceState_Starting))
	assert.False(t, IsHaltOwnedState(ServiceState_Unhealthy))
	assert.False(t, IsHaltOwnedState(ServiceState_Unknown))
}
