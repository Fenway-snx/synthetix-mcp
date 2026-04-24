package transfer

type Type int64

const (
	Type_Unknown Type = iota
	Type_User
	Type_SLPTakeover
	Type_CollateralExchange
)

func (t Type) String() string {
	switch t {
	case Type_Unknown:
		return "unknown"
	case Type_User:
		return "user"
	case Type_SLPTakeover:
		return "slp_takeover"
	case Type_CollateralExchange:
		return "collateral_exchange"
	default:
		return "unknown"
	}
}
