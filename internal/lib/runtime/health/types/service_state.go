package types

import (
	"encoding/json"
	"errors"
	"strings"
)

var (
	errInvalidType = errors.New("invalid type")
)

type IStateReader interface {
	GetServiceState() ServiceState
	GetMetrics() map[string]any
}
type IStateAdviser interface {
	SetServiceState(ServiceState)
	SetMetric(key string, value any) (exists bool, previousValue any)
}

type ServiceState_String string

const (
	ServiceState_Draining_Name     ServiceState_String = "DRAINING"
	ServiceState_Healthy_Name      ServiceState_String = "HEALTHY"
	ServiceState_Idle_Name         ServiceState_String = "IDLE"
	ServiceState_ShuttingDown_Name ServiceState_String = "SHUTTING_DOWN"
	ServiceState_Starting_Name     ServiceState_String = "STARTING"
	ServiceState_Unhealthy_Name    ServiceState_String = "UNHEALTHY"
	ServiceState_Unknown_Name      ServiceState_String = "UNKNOWN"
)

type ServiceState int

// ORDER-CRITICAL ENUMERATION. DO NOT EVER INSERT OR REMOVE ITEMS. You may
// only add new items to the end. Remove items only by renaming them.
const (
	ServiceState_Unknown ServiceState = iota
	ServiceState_Healthy
	ServiceState_ShuttingDown
	ServiceState_Starting
	ServiceState_Unhealthy
	ServiceState_Draining
	ServiceState_Idle
)

func (s ServiceState) String() string {
	switch s {
	case ServiceState_Draining:
		return string(ServiceState_Draining_Name)
	case ServiceState_Healthy:
		return string(ServiceState_Healthy_Name)
	case ServiceState_Idle:
		return string(ServiceState_Idle_Name)
	case ServiceState_ShuttingDown:
		return string(ServiceState_ShuttingDown_Name)
	case ServiceState_Starting:
		return string(ServiceState_Starting_Name)
	case ServiceState_Unhealthy:
		return string(ServiceState_Unhealthy_Name)
	default:
		return string(ServiceState_Unknown_Name)
	}
}

func (s ServiceState) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *ServiceState) UnmarshalJSON(data []byte) error {
	var parsedData any
	if err := json.Unmarshal(data, &parsedData); err != nil {
		return err
	}

	switch v := parsedData.(type) {
	case string:
		s.parseJsonAsString(v)
		return nil

	case float64:
		intVal := int(v)
		if intVal >= int(ServiceState_Unknown) && intVal <= int(ServiceState_Idle) {
			*s = ServiceState(intVal)
		} else {
			*s = ServiceState_Unknown
		}
		return nil

	default:
		return errInvalidType
	}

}

func (s *ServiceState) parseJsonAsString(data string) {
	if strings.EqualFold(data, string(ServiceState_Draining_Name)) {
		*s = ServiceState_Draining
	} else if strings.EqualFold(data, string(ServiceState_Healthy_Name)) {
		*s = ServiceState_Healthy
	} else if strings.EqualFold(data, string(ServiceState_Idle_Name)) {
		*s = ServiceState_Idle
	} else if strings.EqualFold(data, string(ServiceState_ShuttingDown_Name)) {
		*s = ServiceState_ShuttingDown
	} else if strings.EqualFold(data, string(ServiceState_Starting_Name)) {
		*s = ServiceState_Starting
	} else if strings.EqualFold(data, string(ServiceState_Unhealthy_Name)) {
		*s = ServiceState_Unhealthy
	} else {
		*s = ServiceState_Unknown
	}
}

// Returns whether state transitions are owned by halt flow.
func IsHaltOwnedState(state ServiceState) bool {
	return state == ServiceState_Draining ||
		state == ServiceState_Idle ||
		state == ServiceState_ShuttingDown
}
