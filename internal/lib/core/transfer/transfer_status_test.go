package transfer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Status_String(t *testing.T) {
	t.Run("failure status", func(t *testing.T) {
		assert.Equal(t, "failure", Status_Failure.String())
	})

	t.Run("success status", func(t *testing.T) {
		assert.Equal(t, "success", Status_Success.String())
	})

	t.Run("unknown status", func(t *testing.T) {
		assert.Equal(t, "unknown", Status_Unknown.String())
	})

	t.Run("invalid status returns unknown", func(t *testing.T) {
		invalidStatus := Status(999)
		assert.Equal(t, "unknown", invalidStatus.String())
	})

	t.Run("negative status returns unknown", func(t *testing.T) {
		negativeStatus := Status(-1)
		assert.Equal(t, "unknown", negativeStatus.String())
	})
}

func Test_Status_Constants(t *testing.T) {
	t.Run("iota values are sequential", func(t *testing.T) {
		assert.Equal(t, Status(0), Status_Unknown)
		assert.Equal(t, Status(1), Status_Failure)
		assert.Equal(t, Status(2), Status_Success)
	})
}
