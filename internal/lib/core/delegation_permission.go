package core

type DelegationPermission string

const (
	DelegationPermissionDelegate DelegationPermission = "delegate"
	DelegationPermissionSession  DelegationPermission = "session"
	DelegationPermissionTrading  DelegationPermission = "trading"
)

func (p DelegationPermission) String() string {
	return string(p)
}

func (p DelegationPermission) IsSessionLevel() bool {
	return p == DelegationPermissionSession || p == DelegationPermissionTrading
}

func (p DelegationPermission) IsDelegateLevel() bool {
	return p == DelegationPermissionDelegate
}

func PermissionSatisfiedBy(requested, stored DelegationPermission) bool {
	if stored.IsDelegateLevel() {
		return requested.IsDelegateLevel() || requested.IsSessionLevel()
	}
	if stored.IsSessionLevel() {
		return requested.IsSessionLevel()
	}
	return false
}

func IsValidDelegationPermission(p DelegationPermission) bool {
	return p == DelegationPermissionDelegate || p == DelegationPermissionSession || p == DelegationPermissionTrading
}
