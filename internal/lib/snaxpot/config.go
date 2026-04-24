package snaxpot

import (
	"errors"
	"fmt"
	"os"

	shopspring_decimal "github.com/shopspring/decimal"
)

const (
	// Ball ranges mirror the on-chain game definition and must not be changed
	// without a corresponding smart-contract upgrade.
	SnaxBallMax     = 5
	SnaxBallMin     = 1
	StandardBallMax = 32
	StandardBallMin = 1

	// standardBallCount is the number of standard balls drawn per ticket.
	standardBallCount = 5
)

var (
	errSnaxpotEnvVarRequired             = errors.New("snaxpot environment variable is required")
	errSnaxpotUSDPerTicketMustBePositive = errors.New("snaxpot usd per ticket must be greater than 0")
)

// Config holds the runtime-configurable Snaxpot rules. Ball ranges are defined
// as package constants (SnaxBallMin/Max, StandardBallMin/Max) because they are
// fixed by the on-chain game definition and must not drift between deployments.
// Only the ticket price is operator-configurable via SNX_SNAXPOT_USD_PER_TICKET.
type Config struct {
	USDPerTicket shopspring_decimal.Decimal
}

// LoadConfigFromEnv loads the Snaxpot ticket price from
// SNX_SNAXPOT_USD_PER_TICKET. Ball ranges are compiled-in constants.
func LoadConfigFromEnv() (Config, error) {
	usdPerTicket, err := loadRequiredDecimalEnv("SNX_SNAXPOT_USD_PER_TICKET")
	if err != nil {
		return Config{}, err
	}

	cfg := Config{USDPerTicket: usdPerTicket}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate reports whether the ticket price is valid.
func (c Config) Validate() error {
	if c.USDPerTicket.Sign() <= 0 {
		return errSnaxpotUSDPerTicketMustBePositive
	}

	return nil
}

func loadRequiredDecimalEnv(envVar string) (shopspring_decimal.Decimal, error) {
	value := os.Getenv(envVar)
	if value == "" {
		return shopspring_decimal.Zero, fmt.Errorf("%w: %s", errSnaxpotEnvVarRequired, envVar)
	}

	parsedValue, err := shopspring_decimal.NewFromString(value)
	if err != nil {
		return shopspring_decimal.Zero, fmt.Errorf(
			"%s must be a valid decimal: %w",
			envVar,
			err,
		)
	}

	return parsedValue, nil
}
