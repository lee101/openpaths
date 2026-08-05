package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

var (
	siteEntryRe = regexp.MustCompile(`(?s)\{\s*\n\s*id: '([^']+)',(.*?)\n  \}`)
	siteInputRe = regexp.MustCompile(`priceInput: (-?[0-9.]+)`)
	siteOutRe   = regexp.MustCompile(`priceOutput: (-?[0-9.]+)`)
)

// TestSitePricesMatchBilledPrices guards the gap that shipped Claude Fable 5 at
// Opus prices and nvidia/deepseek-v4-pro as free: config.yaml is what the
// gateway bills, src/data/models.ts is only what the site advertises, and
// nothing tied the two together. Advertising less than we bill is the costly
// direction, so any mismatch fails.
func TestSitePricesMatchBilledPrices(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// Asset-priced models (image/video/3D/audio/forecast) bill per image,
	// second, request, etc., and the site puts that per-asset number in
	// priceInput. Different units, so compare only models billed per token.
	billed := map[string]struct{ in, out float64 }{}
	for _, m := range cfg.Models {
		assetPriced := m.PricePerRequest + m.PricePer1MCharacters + m.PricePerImage + m.PricePerMegapixel +
			m.PriceFirstMegapixel + m.PriceExtraMegapixel + m.PricePerInputImage +
			m.PricePerVideo + m.PricePerSecond + m.PricePerSecondWithVideoInput + m.PricePerInputVideoSecond +
			m.PricePerMinute + m.PricePerHour + m.PricePerForecast
		if assetPriced > 0 || len(m.PricePerImageByResolution) > 0 || len(m.PricePerSecondByResolution) > 0 {
			continue
		}
		if m.InputPricePer1M == 0 && m.OutputPricePer1M == 0 {
			continue
		}
		billed[m.ID] = struct{ in, out float64 }{m.InputPricePer1M, m.OutputPricePer1M}
	}

	raw, err := os.ReadFile(filepath.Join("..", "..", "src", "data", "models.ts"))
	if err != nil {
		t.Fatal(err)
	}

	checked := 0
	for _, entry := range siteEntryRe.FindAllStringSubmatch(string(raw), -1) {
		id, body := entry[1], entry[2]
		want, ok := billed[id]
		if !ok {
			continue
		}
		checked++
		for _, f := range []struct {
			name string
			re   *regexp.Regexp
			want float64
		}{
			{"priceInput", siteInputRe, want.in},
			{"priceOutput", siteOutRe, want.out},
		} {
			m := f.re.FindStringSubmatch(body)
			if m == nil {
				continue
			}
			got, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				t.Errorf("%s: unparseable %s %q", id, f.name, m[1])
				continue
			}
			if diff := got - f.want; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("%s: site advertises %s %v but config.yaml bills %v", id, f.name, got, f.want)
			}
		}
	}
	if checked < 50 {
		t.Fatalf("only matched %d site entries; the models.ts parse likely broke", checked)
	}
	t.Logf("checked %d site entries against billed prices", checked)
}
