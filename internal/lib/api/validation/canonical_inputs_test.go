package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ValidateUpdateLeverageAction_RejectsNonCanonicalSymbol(t *testing.T) {
	t.Parallel()

	action := &UpdateLeverageActionPayload{
		Action:   "updateLeverage",
		Symbol:   "btc-usdt",
		Leverage: "10",
	}

	err := ValidateUpdateLeverageAction(action)
	require.Error(t, err)
	require.Equal(t, "symbol must use canonical uppercase format", err.Error())
}

func Test_ValidateCreateSubaccountAction_RejectsWhitespacePaddedName(t *testing.T) {
	t.Parallel()

	action := &CreateSubaccountActionPayload{
		Action: "createSubaccount",
		Name:   "  desk-1  ",
	}

	err := ValidateCreateSubaccountAction(action)
	require.Error(t, err)
	require.Equal(t, "name must not include leading or trailing whitespace", err.Error())
}

func Test_ValidateAddDelegatedSignerAction_RejectsWhitespacePaddedAddress(t *testing.T) {
	t.Parallel()

	action := &AddDelegatedSignerActionPayload{
		Action:          "addDelegatedSigner",
		DelegateAddress: " 0x1234567890123456789012345678901234567890 ",
		Permissions:     []string{"trading"},
	}

	err := ValidateAddDelegatedSignerAction(action)
	require.Error(t, err)
	require.Equal(t, "walletAddress must not include leading or trailing whitespace", err.Error())
}

func Test_ValidateRemoveDelegatedSignerAction_RejectsWhitespacePaddedAddress(t *testing.T) {
	t.Parallel()

	action := &RemoveDelegatedSignerActionPayload{
		Action:          "removeDelegatedSigner",
		DelegateAddress: " 0x1234567890123456789012345678901234567890 ",
	}

	err := ValidateRemoveDelegatedSignerAction(action)
	require.Error(t, err)
	require.Equal(t, "walletAddress must not include leading or trailing whitespace", err.Error())
}
