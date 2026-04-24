package transfer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Type_String(t *testing.T) {
	t.Run("unknown type", func(t *testing.T) {
		assert.Equal(t, "unknown", Type_Unknown.String())
	})

	t.Run("user type", func(t *testing.T) {
		assert.Equal(t, "user", Type_User.String())
	})

	t.Run("slp takeover type", func(t *testing.T) {
		assert.Equal(t, "slp_takeover", Type_SLPTakeover.String())
	})

	t.Run("collateral exchange type", func(t *testing.T) {
		assert.Equal(t, "collateral_exchange", Type_CollateralExchange.String())
	})

	t.Run("invalid type returns unknown", func(t *testing.T) {
		invalidType := Type(999)
		assert.Equal(t, "unknown", invalidType.String())
	})

	t.Run("negative type returns unknown", func(t *testing.T) {
		negativeType := Type(-1)
		assert.Equal(t, "unknown", negativeType.String())
	})
}
func Test_Type_Constants(t *testing.T) {
	t.Run("iota values are sequential", func(t *testing.T) {
		assert.Equal(t, Type(0), Type_Unknown)
		assert.Equal(t, Type(1), Type_User)
		assert.Equal(t, Type(2), Type_SLPTakeover)
		assert.Equal(t, Type(3), Type_CollateralExchange)
	})
}
