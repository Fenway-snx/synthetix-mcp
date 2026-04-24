package snaxpot

import (
	"errors"
	"fmt"
)

var (
	errSnaxBallSequenceDuplicate = errors.New("snax ball sequence contains duplicate values")
	errSnaxBallSequenceEmpty     = errors.New("snax ball sequence is empty")
	errSnaxBallSequenceInvalid   = errors.New("snax ball sequence contains invalid values")
)

// ValidateSnaxBallSequence validates an ordered set of unique Snax balls.
func ValidateSnaxBallSequence(values []int) error {
	if len(values) == 0 {
		return errSnaxBallSequenceEmpty
	}

	seen := make(map[int]struct{}, len(values))
	for _, value := range values {
		if !ValidSnaxBall(value) {
			return fmt.Errorf("%w: %d", errSnaxBallSequenceInvalid, value)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%w: %d", errSnaxBallSequenceDuplicate, value)
		}

		seen[value] = struct{}{}
	}

	return nil
}

// AssignSnaxBallSequence returns a deterministic round-robin assignment over
// count eligible tickets starting from offset.
func AssignSnaxBallSequence(
	values []int,
	count int,
	offset int64,
) ([]int, error) {
	if err := ValidateSnaxBallSequence(values); err != nil {
		return nil, err
	}
	if count < 0 {
		return nil, fmt.Errorf("count must not be negative: %d", count)
	}
	if offset < 0 {
		return nil, fmt.Errorf("offset must not be negative: %d", offset)
	}
	if count == 0 {
		return []int{}, nil
	}

	assignments := make([]int, 0, count)
	valueCount := int64(len(values))
	nextIndex := offset % valueCount
	for i := 0; i < count; i++ {
		assignments = append(assignments, values[nextIndex])
		nextIndex++
		if nextIndex == valueCount {
			nextIndex = 0
		}
	}

	return assignments, nil
}
