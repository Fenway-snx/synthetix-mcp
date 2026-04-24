package types

import "strings"

type DeploymentMode int

const (
	DeploymentMode_Unknown     DeploymentMode = 0               // Deployment mode is not specified, or unknown
	DeploymentMode_Local       DeploymentMode = 1 << (iota - 1) // Local deployment
	DeploymentMode_Development                                  // Development (aka "dev") deployment
	DeploymentMode_Staging                                      // Staging deployment
	DeploymentMode_Production                                   // Production (aka "prod") deployment
	DeploymentMode_Custom
)

// Constraint for types accepted by ParseDeploymentMode.
type deploymentModeInput interface {
	string | []string
}

// Parses a deployment mode from a string or a slice of strings,
// returning the parsed mode and the index of the element that
// produced it.
//
// When given a string:
//   - accepts permissive forms, including any-case, and widely-used
//     contractions;
//   - empty (or whitespace-only) input returns DeploymentMode_Unknown;
//   - unrecognised input returns DeploymentMode_Custom;
//   - the returned index is always 0.
//
// When given a []string, the elements are considered in order and the
// first non-empty (after trimming) element is evaluated; the rest are
// ignored. The returned index identifies which element was used. If
// every element is empty (or the slice itself is empty),
// DeploymentMode_Unknown is returned with an index of -1.
func ParseDeploymentMode[T deploymentModeInput](input T) (DeploymentMode, int) {
	switch v := any(input).(type) {
	case string:
		return parseNormalized(strings.ToLower(strings.TrimSpace(v))), 0
	case []string:
		for i, s := range v {
			if normalized := strings.ToLower(strings.TrimSpace(s)); normalized != "" {
				return parseNormalized(normalized), i
			}
		}
		return DeploymentMode_Unknown, -1
	default:
		return DeploymentMode_Unknown, -1
	}
}

// Matches an already-normalized (trimmed, lowercased) string to a
// DeploymentMode constant.
func parseNormalized(s string) DeploymentMode {
	switch s {
	case "":
		return DeploymentMode_Unknown
	case "custom":
		return DeploymentMode_Custom
	case "dev", "development":
		return DeploymentMode_Development
	case "local":
		return DeploymentMode_Local
	case "prod", "production":
		return DeploymentMode_Production
	case "staging":
		return DeploymentMode_Staging
	default:
		return DeploymentMode_Custom
	}
}

// Implemented by configurations that expose a resolved deployment mode for
// logging and bootstrap behaviour.
//
// Metadata: snx:order=semantic
type DeploymentModeReporter interface {
	CanonicalString() string // provides the canonical string, if any, that matches the deployment mode
	IsLocal() bool
	IsDevelopment() bool
	IsStaging() bool
	IsProduction() bool
	IsCustom() bool
}

var _ DeploymentModeReporter = DeploymentMode(0)

// Returns the canonical deployment_mode strings used in config and
// structured logs. The result is empty for DeploymentMode_Unknown or any
// unrecognised value.
func (dm DeploymentMode) CanonicalString() string {
	switch dm {
	case DeploymentMode_Local:
		return "local"
	case DeploymentMode_Development:
		return "development"
	case DeploymentMode_Staging:
		return "staging"
	case DeploymentMode_Production:
		return "production"
	case DeploymentMode_Custom:
		return "custom"
	default:
		return ""
	}
}

func (dm DeploymentMode) IsLocal() bool       { return dm == DeploymentMode_Local }
func (dm DeploymentMode) IsDevelopment() bool { return dm == DeploymentMode_Development }
func (dm DeploymentMode) IsStaging() bool     { return dm == DeploymentMode_Staging }
func (dm DeploymentMode) IsProduction() bool  { return dm == DeploymentMode_Production }
func (dm DeploymentMode) IsCustom() bool      { return dm == DeploymentMode_Custom }
