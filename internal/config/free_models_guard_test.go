package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

var (
	freeIDsRe     = regexp.MustCompile(`FREE_MODEL_IDS = \[([^\]]*)\]`)
	quotedRe      = regexp.MustCompile(`'([^']+)'`)
	capDefaultRe  = regexp.MustCompile(`kind: '(\w+)',\s*endpoint: '[^']*',\s*defaultModel: '([^']+)'`)
	siteModelIDRe = regexp.MustCompile(`\n    id: '([^']+)',`)
)

// firstPartyGPU are providers served from app.nz/RunPod-backed GPU capacity.
// A $0 route on any of them would let an empty-balance account burn paid GPU
// (and its RunPod overflow lane, which omniserve restricts to paid/sub tiers).
var firstPartyGPU = map[string]bool{
	"appnz": true, "appnz-qwen": true, "netwrck": true, "manifoldgen": true,
	"cutedsl": true, "gobed": true, "textgenerator": true, "runanywhere": true, "runpod": true,
}

func isPriced(t *testing.T, m *model.ModelConfig) bool {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for k, v := range fields {
		if !strings.Contains(k, "price") {
			continue
		}
		switch x := v.(type) {
		case float64:
			if x != 0 {
				return true
			}
		case map[string]any:
			for _, e := range x {
				if f, ok := e.(float64); ok && f != 0 {
					return true
				}
			}
		}
	}
	return false
}

func readSite(t *testing.T, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestFreeModelsAreExternalAndMatchAPIPages pins the set of $0 models in
// config.yaml to the FREE_MODEL_IDS the /api pages advertise and offer to
// empty-balance accounts, and requires each to be an external-provider route.
func TestFreeModelsAreExternalAndMatchAPIPages(t *testing.T) {
	byName := loadAuditConfig(t)
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	zero := map[string]bool{}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		if !isPriced(t, m) {
			zero[m.ID] = true
			if firstPartyGPU[m.Provider] {
				t.Errorf("%s is $0 on first-party GPU provider %q: empty-balance accounts could reach paid GPU/RunPod overflow", m.ID, m.Provider)
			}
		}
	}

	match := freeIDsRe.FindStringSubmatch(readSite(t, "src/data/capabilities.ts"))
	if match == nil {
		t.Fatal("FREE_MODEL_IDS not found in src/data/capabilities.ts")
	}
	var advertised []string
	for _, q := range quotedRe.FindAllStringSubmatch(match[1], -1) {
		advertised = append(advertised, q[1])
	}
	sort.Strings(advertised)

	var billedFree []string
	for id := range zero {
		billedFree = append(billedFree, id)
	}
	sort.Strings(billedFree)

	if strings.Join(advertised, ",") != strings.Join(billedFree, ",") {
		t.Errorf("FREE_MODEL_IDS %v != $0 models in config.yaml %v", advertised, billedFree)
	}
	for _, id := range advertised {
		m := byName[id]
		if m == nil {
			t.Errorf("free model %s missing from config.yaml", id)
			continue
		}
		if isPriced(t, m) {
			t.Errorf("free model %s is priced in config.yaml", id)
		}
	}
}

// TestFirstPartyGPUModelsArePriced: every model on a first-party GPU provider
// bills a positive amount, so BalanceCheck/prepaidGate always demand credits
// before it can run (appnz stamps X-Omniserve-Tier: paid on those calls).
func TestFirstPartyGPUModelsArePriced(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		if firstPartyGPU[m.Provider] && !isPriced(t, m) {
			t.Errorf("%s (%s) has no price", m.ID, m.Provider)
		}
	}
}

// TestAPIPageDefaultsExist: the model each /api try-it box runs by default must
// exist in the site catalog, and runnable (non-studio) defaults must be routable
// from config.yaml.
func TestAPIPageDefaultsExist(t *testing.T) {
	byName := loadAuditConfig(t)
	site := map[string]bool{}
	for _, m := range siteModelIDRe.FindAllStringSubmatch(readSite(t, "src/data/models.ts"), -1) {
		site[m[1]] = true
	}
	defaults := capDefaultRe.FindAllStringSubmatch(readSite(t, "src/data/capabilities.ts"), -1)
	if len(defaults) < 10 {
		t.Fatalf("expected >=10 capability defaults, parsed %d", len(defaults))
	}
	for _, d := range defaults {
		kind, id := d[1], d[2]
		if kind == "studio" {
			continue
		}
		if !site[id] {
			t.Errorf("default %s (%s) missing from src/data/models.ts", id, kind)
		}
		if byName[id] == nil {
			t.Errorf("default %s (%s) not routable in config.yaml", id, kind)
		}
	}
}
