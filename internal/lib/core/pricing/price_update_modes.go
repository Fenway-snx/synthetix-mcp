package pricing

import (
	"fmt"
	"strings"

	snx_lib_config "github.com/Fenway-snx/synthetix-mcp/internal/lib/config"
)

type PriceUpdateMode int64

const (
	PriceUpdateMode_None     PriceUpdateMode = 0
	PriceUpdateMode_Separate PriceUpdateMode = 1 << iota
	PriceUpdateMode_Batched
	PriceUpdateMode_All PriceUpdateMode = PriceUpdateMode_Batched | PriceUpdateMode_Separate
)

var (
	errfmtInvalidModeSpecifier = "invalid mode specifier '%s'"
)

func ParsePriceUpdateMode(s string) (r PriceUpdateMode, err error) {

	names, err := snx_lib_config.ParseUniqueStrings(s, "|")

	if err == nil {

		if len(names) == 0 {
			r = PriceUpdateMode_None
		} else {

			r = PriceUpdateMode_None

			for _, name := range names {
				name = strings.TrimSpace(name)
				name = strings.ToLower(name)

				switch name {
				case "*":

					r |= PriceUpdateMode_All
				case "none":

				case "separate":

					r |= PriceUpdateMode_Separate
				case "batched":

					r |= PriceUpdateMode_Batched
				default:

					err = fmt.Errorf(errfmtInvalidModeSpecifier, name)

					return
				}
			}
		}
	}

	return
}
