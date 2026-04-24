package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Tier_TableName(t *testing.T) {
	assert.Equal(t, "tiers", Tier{}.TableName())
}

func Test_WalletTier_TableName(t *testing.T) {
	assert.Equal(t, "wallet_tiers", WalletTier{}.TableName())
}
