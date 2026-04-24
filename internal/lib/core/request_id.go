package core

import (
	"errors"
	"strings"
)

type RequestId string

var (
	err_RequestId_Empty = errors.New("request id empty")
)

func NewRequestId(input string) (RequestId, error) {
	if strings.TrimSpace(input) == "" {
		return "", err_RequestId_Empty
	}

	return RequestId(input), nil
}
