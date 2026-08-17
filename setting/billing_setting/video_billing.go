package billing_setting

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"github.com/samber/lo"
)

const (
	VideoResolution480P  = "480p"
	VideoResolution720P  = "720p"
	VideoResolution1080P = "1080p"
)

var videoResolutionAliases = map[string]string{
	"480p":      VideoResolution480P,
	"854x480":   VideoResolution480P,
	"480x854":   VideoResolution480P,
	"832x480":   VideoResolution480P,
	"480x832":   VideoResolution480P,
	"640x480":   VideoResolution480P,
	"480x640":   VideoResolution480P,
	"720p":      VideoResolution720P,
	"1280x720":  VideoResolution720P,
	"720x1280":  VideoResolution720P,
	"1080p":     VideoResolution1080P,
	"1920x1080": VideoResolution1080P,
	"1080x1920": VideoResolution1080P,
	"1792x1024": VideoResolution1080P,
	"1024x1792": VideoResolution1080P,
}

// NormalizeVideoResolution maps exact provider spellings to a billing tier.
// Fuzzy pixel matching is intentionally rejected to prevent underbilling.
func NormalizeVideoResolution(size string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(size))
	value = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)
	value = strings.NewReplacer("×", "x", "*", "x").Replace(value)
	resolution, ok := videoResolutionAliases[value]
	return resolution, ok
}

type VideoResolutionPrices struct {
	P480  float64 `json:"480p,omitempty"`
	P720  float64 `json:"720p,omitempty"`
	P1080 float64 `json:"1080p,omitempty"`
}

func (p VideoResolutionPrices) Price(resolution string) (float64, bool) {
	var price float64
	switch resolution {
	case VideoResolution480P:
		price = p.P480
	case VideoResolution720P:
		price = p.P720
	case VideoResolution1080P:
		price = p.P1080
	default:
		return 0, false
	}
	return price, isValidVideoPrice(price)
}

// PreferredPrice returns the preferred priced tier, then 720p / 480p / 1080p.
// It does not invent a price for an unconfigured tier.
func (p VideoResolutionPrices) PreferredPrice(preferred string) (string, float64, bool) {
	if preferred != "" {
		if price, ok := p.Price(preferred); ok {
			return preferred, price, true
		}
	}
	for _, resolution := range []string{VideoResolution720P, VideoResolution480P, VideoResolution1080P} {
		if resolution == preferred {
			continue
		}
		if price, ok := p.Price(resolution); ok {
			return resolution, price, true
		}
	}
	return "", 0, false
}

func VideoResolutionRequestSize(resolution string) string {
	switch resolution {
	case VideoResolution480P:
		return "854x480"
	case VideoResolution1080P:
		return "1920x1080"
	default:
		return "720x1280"
	}
}

func (p VideoResolutionPrices) Validate(model string) error {
	prices := map[string]float64{
		VideoResolution480P:  p.P480,
		VideoResolution720P:  p.P720,
		VideoResolution1080P: p.P1080,
	}
	configured := 0
	for resolution, price := range prices {
		if price == 0 {
			continue
		}
		if !isValidVideoPrice(price) {
			return fmt.Errorf("model %s video price %s must be a finite number greater than 0", model, resolution)
		}
		configured++
	}
	if configured == 0 {
		return fmt.Errorf("model %s must have at least one video resolution price", model)
	}
	return nil
}

func isValidVideoPrice(price float64) bool {
	return price > 0 && !math.IsNaN(price) && !math.IsInf(price, 0)
}

func GetVideoPrices(model string) (VideoResolutionPrices, bool) {
	prices, ok := billingSetting.VideoPrices[model]
	if !ok || prices.Validate(model) != nil {
		return VideoResolutionPrices{}, false
	}
	return prices, true
}

func GetVideoPrice(model, resolution string) (float64, bool) {
	prices, ok := GetVideoPrices(model)
	if !ok {
		return 0, false
	}
	return prices.Price(resolution)
}

func GetVideoPricesCopy() map[string]VideoResolutionPrices {
	return lo.Assign(billingSetting.VideoPrices)
}

func ValidateVideoPricesJSON(value string) error {
	var rawPrices map[string]map[string]float64
	if err := common.Unmarshal([]byte(value), &rawPrices); err != nil {
		return fmt.Errorf("invalid video prices: %w", err)
	}
	prices := make(map[string]VideoResolutionPrices, len(rawPrices))
	for model, rawTiers := range rawPrices {
		var tiers VideoResolutionPrices
		for resolution, price := range rawTiers {
			if !isValidVideoPrice(price) {
				return fmt.Errorf("model %s video price %s must be a finite number greater than 0", model, resolution)
			}
			switch resolution {
			case VideoResolution480P:
				tiers.P480 = price
			case VideoResolution720P:
				tiers.P720 = price
			case VideoResolution1080P:
				tiers.P1080 = price
			default:
				return fmt.Errorf("model %s has unknown video resolution %s", model, resolution)
			}
		}
		prices[model] = tiers
	}
	for model, tiers := range prices {
		if strings.TrimSpace(model) == "" {
			return fmt.Errorf("video price model name cannot be empty")
		}
		if err := tiers.Validate(model); err != nil {
			return err
		}
	}
	for model, mode := range billingSetting.BillingMode {
		if mode != BillingModeVideo {
			continue
		}
		if _, ok := prices[model]; !ok {
			return fmt.Errorf("model %s uses video billing but has no video prices", model)
		}
	}
	return nil
}

func ValidateBillingModesJSON(value string) error {
	var modes map[string]string
	if err := common.Unmarshal([]byte(value), &modes); err != nil {
		return fmt.Errorf("invalid billing modes: %w", err)
	}
	for model, mode := range modes {
		if strings.TrimSpace(model) == "" {
			return fmt.Errorf("billing mode model name cannot be empty")
		}
		if mode != BillingModeVideo {
			continue
		}
		if _, ok := GetVideoPrices(model); !ok {
			return fmt.Errorf("model %s uses video billing but has incomplete video prices", model)
		}
	}
	return nil
}
