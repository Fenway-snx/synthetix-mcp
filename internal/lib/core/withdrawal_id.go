package core

import "errors"

var (
	err_WithdrawalId_Negative = errors.New("negative withdrawal id")
	err_WithdrawalId_Zero     = errors.New("zero withdrawal id")
)

type OffchainWithdrawalId int64

func NewOffchainWithdrawalId(input int64) (OffchainWithdrawalId, error) {
	if input < 0 {
		return 0, err_WithdrawalId_Negative
	}

	if input == 0 {
		return 0, err_WithdrawalId_Zero
	}

	return OffchainWithdrawalId(input), nil
}

type OnchainWithdrawalId int64

func NewOnchainWithdrawalId(input int64) (OnchainWithdrawalId, error) {
	if input < 0 {
		return 0, err_WithdrawalId_Negative
	}

	if input == 0 {
		return 0, err_WithdrawalId_Zero
	}

	return OnchainWithdrawalId(input), nil
}
