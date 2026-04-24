package transfer

type Status int64

const (
	Status_Unknown Status = iota
	Status_Failure
	Status_Success
)

func (s Status) String() string {
	switch s {
	case Status_Failure:
		return "failure"
	case Status_Success:
		return "success"
	case Status_Unknown:
		return "unknown"
	default:
		return "unknown"
	}
}
