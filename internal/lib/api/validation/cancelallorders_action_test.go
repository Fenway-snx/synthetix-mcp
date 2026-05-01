package validation

import (
	"testing"
)

func Test_ValidateCancelAllOrdersAction_Success(t *testing.T) {
	action := &CancelAllOrdersActionPayload{
		Action:  "cancelAllOrders",
		Symbols: []Symbol{"BTC-USDT", "ETH-USDT"},
	}

	symbols, err := ValidateCancelAllOrdersAction(action)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(symbols) != 2 {
		t.Fatalf("expected 2 symbols, got: %d", len(symbols))
	}

	if symbols[0] != "BTC-USDT" || symbols[1] != "ETH-USDT" {
		t.Fatalf("unexpected symbols: %v", symbols)
	}
}

func Test_ValidateCancelAllOrdersAction_Wildcard(t *testing.T) {
	action := &CancelAllOrdersActionPayload{
		Action:  "cancelAllOrders",
		Symbols: []Symbol{"*"},
	}

	symbols, err := ValidateCancelAllOrdersAction(action)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(symbols) != 1 || symbols[0] != "*" {
		t.Fatalf("expected [\"*\"], got: %v", symbols)
	}
}

func Test_ValidateCancelAllOrdersAction_RejectsWhitespace(t *testing.T) {
	action := &CancelAllOrdersActionPayload{
		Action:  "cancelAllOrders",
		Symbols: []Symbol{"  BTC-USDT  ", " ETH-USDT"},
	}

	_, err := ValidateCancelAllOrdersAction(action)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "symbols[0] must use canonical uppercase format" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_ValidateCancelAllOrdersAction_RejectsNonCanonicalSymbols(t *testing.T) {
	action := &CancelAllOrdersActionPayload{
		Action:  "cancelAllOrders",
		Symbols: []Symbol{" btc-usdt "},
	}

	_, err := ValidateCancelAllOrdersAction(action)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "symbols[0] must use canonical uppercase format" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_ValidateCancelAllOrdersAction_RejectsSymbolsNeedingNormalization(t *testing.T) {
	action := &CancelAllOrdersActionPayload{
		Action:  "cancelAllOrders",
		Symbols: []Symbol{" btc-usdt "},
	}

	_, err := ValidateCancelAllOrdersAction(action)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "symbols[0] must use canonical uppercase format" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_ValidateCancelAllOrdersAction_EmptyAction(t *testing.T) {
	action := &CancelAllOrdersActionPayload{
		Action:  "",
		Symbols: []Symbol{"BTC-USDT"},
	}

	_, err := ValidateCancelAllOrdersAction(action)
	if err != nil {
		t.Fatalf("expected no error for empty action, got: %v", err)
	}
}

func Test_ValidateCancelAllOrdersAction_Errors(t *testing.T) {
	tests := []struct {
		name    string
		action  *CancelAllOrdersActionPayload
		wantErr string
	}{
		{
			name:    "nil action",
			action:  nil,
			wantErr: "action payload is required",
		},
		{
			name: "wrong action type",
			action: &CancelAllOrdersActionPayload{
				Action:  "cancelOrders",
				Symbols: []Symbol{"BTC-USDT"},
			},
			wantErr: "action type must be 'cancelAllOrders'",
		},
		{
			name: "empty symbols array",
			action: &CancelAllOrdersActionPayload{
				Action:  "cancelAllOrders",
				Symbols: []Symbol{},
			},
			wantErr: "symbols must be nonempty",
		},
		{
			name: "nil symbols",
			action: &CancelAllOrdersActionPayload{
				Action:  "cancelAllOrders",
				Symbols: nil,
			},
			wantErr: "symbols must be nonempty",
		},
		{
			name: "wildcard mixed with symbol",
			action: &CancelAllOrdersActionPayload{
				Action:  "cancelAllOrders",
				Symbols: []Symbol{"*", "ETH-USDT"},
			},
			wantErr: `symbols[0]: wildcard "*" must be the only element in symbols`,
		},
		{
			name: "symbol mixed with wildcard",
			action: &CancelAllOrdersActionPayload{
				Action:  "cancelAllOrders",
				Symbols: []Symbol{"BTC-USDT", "*"},
			},
			wantErr: `symbols[1]: wildcard "*" must be the only element in symbols`,
		},
		{
			name: "symbol is empty string",
			action: &CancelAllOrdersActionPayload{
				Action:  "cancelAllOrders",
				Symbols: []Symbol{"BTC-USDT", ""},
			},
			wantErr: "symbols[1]: symbol cannot be empty or whitespace",
		},
		{
			name: "symbol is whitespace only",
			action: &CancelAllOrdersActionPayload{
				Action:  "cancelAllOrders",
				Symbols: []Symbol{"BTC-USDT", "   ", "ETH-USDT"},
			},
			wantErr: "symbols[1]: symbol cannot be empty or whitespace",
		},
		{
			name: "first symbol is empty",
			action: &CancelAllOrdersActionPayload{
				Action:  "cancelAllOrders",
				Symbols: []Symbol{"", "BTC-USDT"},
			},
			wantErr: "symbols[0]: symbol cannot be empty or whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateCancelAllOrdersAction(tt.action)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func Test_NewValidatedCancelAllOrdersAction_Success(t *testing.T) {
	payload := &CancelAllOrdersActionPayload{
		Action:  "cancelAllOrders",
		Symbols: []Symbol{"BTC-USDT", "ETH-USDT"},
	}

	validated, err := NewValidatedCancelAllOrdersAction(payload)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if validated.Payload != payload {
		t.Fatalf("expected payload to match")
	}

	if len(validated.Symbols) != 2 {
		t.Fatalf("expected 2 symbols, got: %d", len(validated.Symbols))
	}
}

func Test_NewValidatedCancelAllOrdersAction_Wildcard(t *testing.T) {
	payload := &CancelAllOrdersActionPayload{
		Action:  "cancelAllOrders",
		Symbols: []Symbol{"*"},
	}

	validated, err := NewValidatedCancelAllOrdersAction(payload)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(validated.Symbols) != 1 || validated.Symbols[0] != "*" {
		t.Fatalf("expected [\"*\"], got: %v", validated.Symbols)
	}
}

func Test_NewValidatedCancelAllOrdersAction_EmptySymbols(t *testing.T) {
	payload := &CancelAllOrdersActionPayload{
		Action:  "cancelAllOrders",
		Symbols: []Symbol{},
	}

	_, err := NewValidatedCancelAllOrdersAction(payload)
	if err == nil {
		t.Fatalf("expected error for empty symbols, got nil")
	}

	if err.Error() != "symbols must be nonempty" {
		t.Fatalf("expected 'symbols must be nonempty', got: %v", err)
	}
}
