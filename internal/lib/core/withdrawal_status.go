package core

type WithdrawalStatus int64

/*
Never update this "enum" without checking the deposit contract enum(on chain)
Types that start at 100 are custom and only used off chain.
We need to watch out for the zero value but I am not sure how we can do it
as it will require updates on the deposit contract.
*/
const (
	WithdrawalStatus_RequestedOnChain   WithdrawalStatus = 0
	WithdrawalStatus_Validated          WithdrawalStatus = 1
	WithdrawalStatus_Disbursed          WithdrawalStatus = 2
	WithdrawalStatus_Denied             WithdrawalStatus = 3
	WithdrawalStatus_Disputed           WithdrawalStatus = 4
	WithdrawalStatus_Cancelled          WithdrawalStatus = 5
	WithdrawalStatus_Expired            WithdrawalStatus = 6
	WithdrawalStatus_Completed          WithdrawalStatus = 100
	WithdrawalStatus_Created            WithdrawalStatus = 101
	WithdrawalStatus_Failed             WithdrawalStatus = 102
	WithdrawalStatus_WaitingActorUpdate WithdrawalStatus = 103
)

func (s WithdrawalStatus) String() string {
	switch s {
	case WithdrawalStatus_Cancelled:
		return "cancelled"
	case WithdrawalStatus_Completed:
		return "completed"
	case WithdrawalStatus_Created:
		return "created"
	case WithdrawalStatus_Denied:
		return "denied"
	case WithdrawalStatus_Disbursed:
		return "disbursed"
	case WithdrawalStatus_Disputed:
		return "disputed"
	case WithdrawalStatus_Expired:
		return "expired"
	case WithdrawalStatus_Failed:
		return "failed"
	case WithdrawalStatus_RequestedOnChain:
		return "requested on chain"
	case WithdrawalStatus_Validated:
		return "validated"
	case WithdrawalStatus_WaitingActorUpdate:
		return "waiting for actor update"
	default:
		return "unknown"
	}
}
