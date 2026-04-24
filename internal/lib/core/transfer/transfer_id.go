package transfer

import "errors"

const (
	IdZero = Id(0)
)

var (
	errInvalidId = errors.New("invalid transfer id")
)

type Id int64

func NewId(input int64) (Id, error) {
	if input < 1 {
		return 0, errInvalidId
	}

	return Id(input), nil
}
