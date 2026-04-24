// Definition of `PerSubAccountRateLimiterConfig`,
// `LoadOrderRateLimiterConfig()` and supporting functions.
//
// NOTE: Much of the supporting functionality needs to be moved into a
// common conversion/config area.
package ratelimiting

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	angols_strings "github.com/synesissoftware/ANGoLS/strings"

	snx_lib_config "github.com/Fenway-snx/synthetix-mcp/internal/lib/config"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

const (
	keyPrefix = "ratelimiting"

	keyGeneralRateLimit    = keyPrefix + ".general_rate_limit"
	keyHandlerTokenCosts   = keyPrefix + ".handler_token_costs"
	keyIPHandlerTokenCosts = keyPrefix + ".ip_handler_token_costs"
	keyIPRateLimit         = keyPrefix + ".ip_rate_limit"
	keySpecificRateLimits  = keyPrefix + ".specific_rate_limits"
	keyWindowMs            = keyPrefix + ".window_ms"
)

// Configuration to be obtained, at load time, for use with the API function
// `NewPerIPRateLimiterFromConfig()`.
type PerIPRateLimiterConfig struct {
	WindowMs    int64     // config: "window_ms"
	IPRateLimit RateLimit // config: "ip_rate_limit"
}

// Configuration to be obtained, at load time, for use with the API function
// `NewPerSubAccountRateLimiterFromConfig()`.
type PerSubAccountRateLimiterConfig struct {
	WindowMs           int64                   // config: "window_ms"
	GeneralRateLimit   RateLimit               // config: "general_rate_limit"
	SpecificRateLimits PerSubAccountRateLimits // config: "specific_rate_limits"
}

// Loads specific rate limits either from an unsplit string or from a map.
func loadSpecificRateLimits(
	v *viper.Viper,
	key string,
) (
	specificRateLimits PerSubAccountRateLimits,
	err error,
) {
	srl_a := v.Get(key)

	if srl_a == nil {
		return
	}

	switch v := srl_a.(type) {
	case map[string]any:
		specificRateLimits, err = mapToSpecifics(v)
	case map[string]string:
		specificRateLimits, err = mapToSpecifics(v)
	case string:
		specificRateLimits, err = stringToSpecifics(v)
	case fmt.Stringer:
		specificRateLimits, err = stringToSpecifics(v.String())
	default:
		err = errUnrecognisedType
	}

	return
}

func mapToSpecifics[V any | string](
	m map[string]V,
) (
	specificRateLimits PerSubAccountRateLimits,
	err error,
) {
	specificRateLimits = make(PerSubAccountRateLimits)

	for k, v := range m {
		// NOTE: keys are unsigned and values are signed

		var k_i int64
		k_i, err = strconv.ParseInt(k, 10, 64)
		if err != nil {
			return
		}

		var v_i int64
		var v_s string
		var isString bool

		switch v_t := any(v).(type) {
		// unsigned integers
		case uint64:
			if v_t > math.MaxInt64 {
				err = errRateLimitsMayNotBeNagative

				return
			} else {
				v_i = int64(v_t)
			}
		case uint32:
			v_i = int64(v_t)
		case uint16:
			v_i = int64(v_t)
		case uint8:
			v_i = int64(v_t)
		case uint:
			v_i = int64(v_t)
		// signed integers
		case int64:
			v_i = int64(v_t)
		case int32:
			v_i = int64(v_t)
		case int16:
			v_i = int64(v_t)
		case int8:
			v_i = int64(v_t)
		case int:
			v_i = int64(v_t)
		// string(s)
		case string:
			v_s = v_t
			isString = true
		case fmt.Stringer:
			v_s = v_t.String()
			isString = true
		// others - can't handle
		default:
			err = errUnrecognisedType

			return
		}

		if isString {
			v_i, err = strconv.ParseInt(v_s, 10, 64)
			if err != nil {
				return
			}
		}

		if v_i < 0 {
			err = errRateLimitsMayNotBeNagative

			return
		}

		specificRateLimits[snx_lib_core.SubAccountId(k_i)] = RateLimit(v_i)
	}

	return
}

// TODO: place parts of this in a common area
func stringToSpecifics(s string) (r PerSubAccountRateLimits, err error) {
	s = strings.TrimSpace(s)

	pairs := strings.Split(s, ",")

	r = make(PerSubAccountRateLimits)

	for i := 0; i != len(pairs); i++ {
		pair := strings.TrimSpace(pairs[i])

		if len(pair) == 0 {
			continue
		}

		kv := strings.Split(pair, "=")

		if 2 != len(kv) {
			err = errInvalidKeyValuePair

			return
		}

		var k int64
		var v int64

		k, err = strconv.ParseInt(kv[0], 10, 64)
		if err != nil {
			return
		}

		v, err = strconv.ParseInt(kv[1], 10, 64)
		if err != nil {
			return
		}

		if v < 0 {
			err = errRateLimitsMayNotBeNagative

			return
		}

		r[snx_lib_core.SubAccountId(k)] = RateLimit(v)
	}

	return r, nil
}

func LoadIPRateLimiterConfig(v *viper.Viper) (cfg PerIPRateLimiterConfig, err error) {
	windowMS := snx_lib_config.GetInt64OrDefault(v, keyWindowMs, defaultWindow.Milliseconds())
	if windowMS < 1 {
		err = errRateLimitDurationMustBePositive

		return
	}

	ipRateLimit := snx_lib_config.GetInt64OrDefault(v, keyIPRateLimit, defaultIPRateLimit)
	if ipRateLimit < 0 {
		err = errRateLimitsMayNotBeNagative

		return
	}

	cfg = PerIPRateLimiterConfig{
		WindowMs:    windowMS,
		IPRateLimit: RateLimit(ipRateLimit),
	}

	return
}

// TokenCost represents the token cost of a single handler invocation.
type TokenCost = int

// HandlerTokenCosts maps request action names (e.g. "placeOrders",
// "getPositions") to their per-request token cost. When a handler is not
// present in the map, DefaultHandlerTokenCost is used.
type HandlerTokenCosts map[RequestAction]TokenCost

// lookupTokenCostWithDefault returns the token cost for the given action. If
// the action is not present in the map, defaultCost is returned.
func (h HandlerTokenCosts) lookupTokenCostWithDefault(action RequestAction, defaultCost TokenCost) TokenCost {
	if cost, ok := h[action]; ok {
		return cost
	}

	return defaultCost
}

// LookupTokenCost returns the total token cost for the given action scaled
// by batchSize. If the action is not present in the map,
// DefaultHandlerTokenCost is used as the base cost. batchSize is clamped
// to a minimum of 1 so that every request consumes at least one unit of
// the base cost.
func (h HandlerTokenCosts) LookupTokenCost(action RequestAction, batchSize int) TokenCost {
	return h.lookupTokenCostWithDefault(action, DefaultHandlerTokenCost) * max(1, batchSize)
}

// LookupIPTokenCost returns the total IP token cost for the given action
// scaled by batchSize. If the action is not present in the map,
// DefaultIPHandlerTokenCost is used as the base cost. batchSize is clamped
// to a minimum of 1 so that every request consumes at least one unit of
// the base cost.
func (h HandlerTokenCosts) LookupIPTokenCost(action RequestAction, batchSize int) TokenCost {
	return h.lookupTokenCostWithDefault(action, DefaultIPHandlerTokenCost) * max(1, batchSize)
}

// loadHandlerTokenCostsFromKey loads per-handler token costs from the viper
// config key. The env var format is "action1=cost1,action2=cost2,...".
// Returns an empty (non-nil) map when the key is not set.
func loadHandlerTokenCostsFromKey(v *viper.Viper, key string) (costs HandlerTokenCosts, err error) {
	raw := v.Get(key)

	if raw == nil {
		costs = HandlerTokenCosts{}

		return
	}

	var s string
	switch v := raw.(type) {
	case string:
		s = v
	case fmt.Stringer:
		s = v.String()
	default:
		err = errUnrecognisedType

		return
	}

	costs, err = parseHandlerTokenCosts(s)

	return
}

// LoadHandlerTokenCosts loads per-handler token costs from the viper config.
func LoadHandlerTokenCosts(v *viper.Viper) (HandlerTokenCosts, error) {
	return loadHandlerTokenCostsFromKey(v, keyHandlerTokenCosts)
}

// LoadIPHandlerTokenCosts loads per-handler token costs for IP rate limiting
// from the viper config.
func LoadIPHandlerTokenCosts(v *viper.Viper) (HandlerTokenCosts, error) {
	return loadHandlerTokenCostsFromKey(v, keyIPHandlerTokenCosts)
}

// parseHandlerTokenCosts parses the "action1=cost1,action2=cost2,..." format.
func parseHandlerTokenCosts(s string) (costs HandlerTokenCosts, err error) {
	s = strings.TrimSpace(s)

	const flags = angols_strings.ParseKeyValuePairsListOption_IgnoreAnonymousValue |
		angols_strings.ParseKeyValuePairsListOption_IgnoreValuelessKey |
		angols_strings.ParseKeyValuePairsListOption_TakeFirstRepeatedKey |
		angols_strings.ParseKeyValuePairsListOption_PreserveOrder

	pairs, err := angols_strings.ParseKeyValuePairsList(
		s,
		",",
		"=",
		(flags),
	)
	if err != nil {
		return
	}

	costs = make(HandlerTokenCosts, len(pairs))

	for _, kv := range pairs {
		var cost int64
		cost, err = strconv.ParseInt(strings.TrimSpace(kv.Value), 10, 32)
		if err != nil {
			return
		}

		if cost < 0 {
			err = errRateLimitsMayNotBeNagative

			return
		}

		costs[RequestAction(strings.TrimSpace(kv.Key))] = int(cost)
	}

	return
}

func LoadOrderRateLimiterConfig(v *viper.Viper) (cfg PerSubAccountRateLimiterConfig, err error) {
	windowMS := snx_lib_config.GetInt64OrDefault(v, keyWindowMs, defaultWindow.Milliseconds())
	if windowMS < 1 {
		err = errRateLimitDurationMustBePositive

		return
	}

	generalRateLimit := snx_lib_config.GetInt64OrDefault(v, keyGeneralRateLimit, defaultOrderRateLimit)
	if generalRateLimit < 0 {
		err = errRateLimitsMayNotBeNagative

		return
	}

	specificRateLimits, err := loadSpecificRateLimits(v, keySpecificRateLimits)
	if err != nil {
		return
	}

	if specificRateLimits == nil {
		specificRateLimits = PerSubAccountRateLimits{}
	}

	cfg = PerSubAccountRateLimiterConfig{
		WindowMs:           windowMS,
		GeneralRateLimit:   RateLimit(generalRateLimit),
		SpecificRateLimits: specificRateLimits,
	}

	return
}
