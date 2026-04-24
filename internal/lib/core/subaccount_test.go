package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_PermissionSatisfiedBy(t *testing.T) {
	tests := []struct {
		name      string
		requested DelegationPermission
		stored    DelegationPermission
		expected  bool
	}{
		// delegate stored satisfies all valid requests
		{"delegate satisfies delegate", DelegationPermissionDelegate, DelegationPermissionDelegate, true},
		{"delegate satisfies session", DelegationPermissionSession, DelegationPermissionDelegate, true},
		{"delegate satisfies trading", DelegationPermissionTrading, DelegationPermissionDelegate, true},

		// session stored satisfies session-level only
		{"session satisfies session", DelegationPermissionSession, DelegationPermissionSession, true},
		{"session satisfies trading", DelegationPermissionTrading, DelegationPermissionSession, true},
		{"session does NOT satisfy delegate", DelegationPermissionDelegate, DelegationPermissionSession, false},

		// trading stored satisfies session-level only (legacy alias)
		{"trading satisfies session", DelegationPermissionSession, DelegationPermissionTrading, true},
		{"trading satisfies trading", DelegationPermissionTrading, DelegationPermissionTrading, true},
		{"trading does NOT satisfy delegate", DelegationPermissionDelegate, DelegationPermissionTrading, false},

		// admin stored satisfies nothing (not a valid delegation tier)
		{"admin does NOT satisfy delegate", DelegationPermissionDelegate, DelegationPermissionAdmin, false},
		{"admin does NOT satisfy session", DelegationPermissionSession, DelegationPermissionAdmin, false},
		{"admin does NOT satisfy trading", DelegationPermissionTrading, DelegationPermissionAdmin, false},

		// unknown stored satisfies nothing
		{"unknown does NOT satisfy session", DelegationPermissionSession, DelegationPermission("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PermissionSatisfiedBy(tt.requested, tt.stored)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_IsSessionLevel(t *testing.T) {
	assert.True(t, DelegationPermissionSession.IsSessionLevel())
	assert.True(t, DelegationPermissionTrading.IsSessionLevel())
	assert.False(t, DelegationPermissionDelegate.IsSessionLevel())
	assert.False(t, DelegationPermissionAdmin.IsSessionLevel())
}

func Test_IsDelegateLevel(t *testing.T) {
	assert.True(t, DelegationPermissionDelegate.IsDelegateLevel())
	assert.False(t, DelegationPermissionSession.IsDelegateLevel())
	assert.False(t, DelegationPermissionTrading.IsDelegateLevel())
	assert.False(t, DelegationPermissionAdmin.IsDelegateLevel())
}

func Test_IsValidDelegationPermission(t *testing.T) {
	assert.True(t, IsValidDelegationPermission(DelegationPermissionDelegate))
	assert.True(t, IsValidDelegationPermission(DelegationPermissionSession))
	assert.True(t, IsValidDelegationPermission(DelegationPermissionTrading))
	assert.False(t, IsValidDelegationPermission(DelegationPermissionAdmin))
	assert.False(t, IsValidDelegationPermission(DelegationPermission("unknown")))
	assert.False(t, IsValidDelegationPermission(DelegationPermission("")))
}
