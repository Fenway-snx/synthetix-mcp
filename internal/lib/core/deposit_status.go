package core

type DepositStatus int64

const (
	DepositStatus_Failed  DepositStatus = 0
	DepositStatus_Success DepositStatus = 1
)

func (s DepositStatus) String() string {
	switch s {
	case DepositStatus_Failed:
		return "failed"
	case DepositStatus_Success:
		return "success"
	default:
		return "unknown"
	}
}
