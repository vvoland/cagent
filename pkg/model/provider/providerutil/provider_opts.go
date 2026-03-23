package providerutil

import "math"

// GetProviderOptFloat64 extracts a float64 value from provider opts.
// YAML may parse numbers as float64 or int, so this handles both.
func GetProviderOptFloat64(opts map[string]any, key string) (float64, bool) {
	if opts == nil {
		return 0, false
	}
	v, ok := opts[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// GetProviderOptInt64 extracts an int64 value from provider opts.
// YAML may parse numbers as float64 or int, so this handles both.
func GetProviderOptInt64(opts map[string]any, key string) (int64, bool) {
	if opts == nil {
		return 0, false
	}
	v, ok := opts[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		if n == math.Trunc(n) && n >= math.MinInt64 && n <= math.MaxInt64 {
			return int64(n), true
		}
		return 0, false
	default:
		return 0, false
	}
}

// samplingProviderOptsKeys lists the provider_opts keys that are
// treated as sampling parameters and forwarded to provider APIs.
// Provider-specific infrastructure keys (api_type, transport, region, etc.)
// are NOT included here.
var samplingProviderOptsKeys = []string{
	"top_k",
	"repetition_penalty",
	"seed",
	"min_p",
	"typical_p",
}

// SamplingProviderOptsKeys returns the list of provider_opts keys that are
// treated as sampling parameters and forwarded to provider APIs.
func SamplingProviderOptsKeys() []string {
	return append([]string(nil), samplingProviderOptsKeys...)
}
