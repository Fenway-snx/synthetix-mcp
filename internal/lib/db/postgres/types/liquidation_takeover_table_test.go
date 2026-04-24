package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Liquidation_TableName(t *testing.T) {
	assert.Equal(t, "liquidations", Liquidation{}.TableName())
}

func Test_SubaccountTakeover_TableName(t *testing.T) {
	assert.Equal(t, "subaccount_takeovers", SubaccountTakeover{}.TableName())
}
