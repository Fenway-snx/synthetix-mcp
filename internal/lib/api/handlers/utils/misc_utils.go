package utils

import (
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
)

const (
	invalidStartTime = "invalid start time"
	invalidEndTime   = "invalid end time"
)

var (
	errInvalidNowTime = errors.New("invalid now time")
)

// Resolves the canonical startTime/endTime from both the new
// (startTime/endTime) and deprecated (fromTime/toTime) parameter pairs.
// Returns an error if both the new and deprecated parameter for the same bound
// are provided with different non-zero values.
func CoalesceTimeRange(startTime, endTime, fromTime, toTime Timestamp) (Timestamp, Timestamp, error) {
	resolvedStart, err := coalesceTimestamp(startTime, fromTime, "startTime", "fromTime")
	if err != nil {
		return Timestamp_Zero, Timestamp_Zero, err
	}

	resolvedEnd, err := coalesceTimestamp(endTime, toTime, "endTime", "toTime")
	if err != nil {
		return Timestamp_Zero, Timestamp_Zero, err
	}

	return resolvedStart, resolvedEnd, nil
}

func coalesceTimestamp(primary, deprecated Timestamp, primaryName, deprecatedName string) (Timestamp, error) {
	if primary != Timestamp_Zero && deprecated != Timestamp_Zero && primary != deprecated {
		return Timestamp_Zero, fmt.Errorf("cannot specify both %s and %s with different values", primaryName, deprecatedName)
	}
	if primary != Timestamp_Zero {
		return primary, nil
	}
	return deprecated, nil
}

// Utility function to obtain timestamppb.Timestamp for start and end
// pointers. It is implemented in terms of TimestampToTimestampPB(), and so
// fails on invalid inputs, but
// has additional semantics:
//   - when either value is Timestamp_Zero then its corresponding pointer is
//     nil;
//   - for a valid end-time, it is capped to now;
func APIStartEndToCoreStartEndPtrs(
	startTime Timestamp,
	endTime Timestamp,
	now Timestamp,
) (
	startTimePtr *timestamppb.Timestamp,
	endTimePtr *timestamppb.Timestamp,
	err error,
	failureQualifier string,
) {
	if !now.IsValid() {
		err = errInvalidNowTime

		return
	}

	if startTime != Timestamp_Zero {
		startTimePtr, err = snx_lib_api_types.TimestampToTimestampPB(startTime)
		if err != nil {

			failureQualifier = invalidStartTime

			return
		}
	}

	if endTime != Timestamp_Zero {
		if endTime.IsValid() && endTime > now {

			endTime = now
		}

		endTimePtr, err = snx_lib_api_types.TimestampToTimestampPB(endTime)
		if err != nil {

			failureQualifier = invalidEndTime

			return
		}
	}

	return
}

// Maps an internally derived trade direction string to the canonical API
// side value ("buy" or "sell"). Returns an error for any direction value
// that is not recognized, rather than silently defaulting.
//
// Recognized sell directions: "open short", "close long", "short", "sell"
// Recognized buy  directions: "open long",  "close short", "long",  "buy"
func TradeDirectionToSide(direction string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(direction)) {
	case "open short", "close long", "short", "sell":
		return "sell", nil
	case "open long", "close short", "long", "buy":
		return "buy", nil
	default:
		return "", fmt.Errorf("unrecognized trade direction: %q", direction)
	}
}
