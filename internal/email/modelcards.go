package email

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
)

// modelNote is one editable entry in emails/model_notes.json. Cards render
// only for models present in the live catalog, with live pricing/context, so
// emails never name stale or removed models.
type modelNote struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Note     string `json:"note"`
	Settings string `json:"settings,omitempty"`
}

type modelNotes struct {
	Sections map[string][]modelNote `json:"sections"`
}

// renderModelCards returns section -> HTML card list for template injection
// via {{.ModelCards.<section>}}.
func renderModelCards(notesPath string, models []model.ModelConfig) (map[string]string, error) {
	data, err := os.ReadFile(notesPath)
	if err != nil {
		return nil, err
	}
	var notes modelNotes
	if err := json.Unmarshal(data, &notes); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", notesPath, err)
	}

	byID := make(map[string]*model.ModelConfig, len(models)*3)
	for i := range models {
		m := &models[i]
		byID[strings.ToLower(m.ID)] = m
		for _, a := range m.Aliases {
			byID[strings.ToLower(a)] = m
		}
	}

	out := make(map[string]string, len(notes.Sections))
	for section, entries := range notes.Sections {
		var b strings.Builder
		for _, n := range entries {
			m, ok := byID[strings.ToLower(n.ID)]
			if !ok {
				continue
			}
			b.WriteString(renderCard(n, m))
		}
		out[section] = b.String()
	}
	return out, nil
}

func priceLine(m *model.ModelConfig) string {
	price := ""
	switch {
	case m.InputPricePer1M > 0 || m.OutputPricePer1M > 0:
		price = fmt.Sprintf("$%s/M in · $%s/M out", trimFloat(m.InputPricePer1M), trimFloat(m.OutputPricePer1M))
	case m.PricePerImage > 0:
		price = fmt.Sprintf("$%s/image", trimFloat4(m.PricePerImage))
	case m.PricePerSecond > 0:
		price = fmt.Sprintf("$%s/second", trimFloat4(m.PricePerSecond))
	case m.PricePerVideo > 0:
		price = fmt.Sprintf("$%s/video", trimFloat4(m.PricePerVideo))
	case m.PricePerMinute > 0:
		price = fmt.Sprintf("$%s/minute", trimFloat4(m.PricePerMinute))
	case m.PricePerRequest > 0:
		price = fmt.Sprintf("$%s/request", trimFloat4(m.PricePerRequest))
	}
	if m.ContextWindow > 0 {
		ctx := fmt.Sprintf("%dK ctx", m.ContextWindow/1000)
		if m.ContextWindow >= 1000000 {
			ctx = fmt.Sprintf("%.1fM ctx", float64(m.ContextWindow)/1e6)
		}
		if price != "" {
			price += " · " + ctx
		} else {
			price = ctx
		}
	}
	return price
}

func renderCard(n modelNote, m *model.ModelConfig) string {
	price := priceLine(m)
	settings := ""
	if n.Settings != "" {
		settings = fmt.Sprintf(`<p style="margin:6px 0 0; color:#6a9a8a; font-size:12px; line-height:1.5; font-family:monospace;">%s</p>`, n.Settings)
	}
	return fmt.Sprintf(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="margin-bottom:8px;">
<tr><td style="padding:14px; background:#1a1a1a; border:1px solid #2a2a2a; border-radius:8px;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0"><tr>
<td>
<p style="margin:0 0 4px; color:#e0e0e0; font-size:14px; font-weight:600;">%s <code style="color:#888; background:#0a0a0a; padding:1px 5px; border-radius:3px; font-size:11px;">%s</code></p>
<p style="margin:0; color:#888; font-size:13px; line-height:1.5;">%s</p>
%s
</td>
<td style="text-align:right; white-space:nowrap; padding-left:16px; vertical-align:top;">
<p style="margin:0; color:#f5f5f5; font-size:12px;">%s</p>
</td>
</tr></table>
</td></tr>
</table>
`, n.Name, m.ID, n.Note, settings, price)
}

func trimFloat(f float64) string {
	s := fmt.Sprintf("%.2f", f)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

func trimFloat4(f float64) string {
	s := fmt.Sprintf("%.4f", f)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

func FindModel(models []model.ModelConfig, id string) *model.ModelConfig {
	id = strings.ToLower(strings.TrimSpace(id))
	for i := range models {
		if strings.ToLower(models[i].ID) == id {
			return &models[i]
		}
	}
	for i := range models {
		for _, a := range models[i].Aliases {
			if strings.ToLower(a) == id {
				return &models[i]
			}
		}
	}
	return nil
}

type ModelCard struct {
	ID      string
	Name    string
	Price   string
	Context string
	HTML    string
}

func BuildModelCard(notesPath string, m *model.ModelConfig) ModelCard {
	note := modelNote{ID: m.ID, Name: m.ID, Note: defaultNote(m)}
	if data, err := os.ReadFile(notesPath); err == nil {
		var notes modelNotes
		if json.Unmarshal(data, &notes) == nil {
		found:
			for _, entries := range notes.Sections {
				for _, n := range entries {
					if FindModel([]model.ModelConfig{*m}, n.ID) != nil {
						note = n
						break found
					}
				}
			}
		}
	}
	ctxLine := ""
	if m.ContextWindow >= 1000000 {
		ctxLine = fmt.Sprintf("%.1fM", float64(m.ContextWindow)/1e6)
	} else if m.ContextWindow > 0 {
		ctxLine = fmt.Sprintf("%dK", m.ContextWindow/1000)
	}
	return ModelCard{ID: m.ID, Name: note.Name, Price: priceLine(m), Context: ctxLine, HTML: renderCard(note, m)}
}

func defaultNote(m *model.ModelConfig) string {
	var caps []string
	if m.SupportsTools {
		caps = append(caps, "tool use")
	}
	if m.SupportsVision {
		caps = append(caps, "vision")
	}
	if m.SupportsStreaming {
		caps = append(caps, "streaming")
	}
	s := "Now routable on OpenPaths through the same API key and OpenAI-compatible endpoint you already use."
	if len(caps) > 0 {
		s += " Supports " + strings.Join(caps, ", ") + "."
	}
	return s
}
