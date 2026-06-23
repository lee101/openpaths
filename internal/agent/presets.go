package agent

import "github.com/openpaths/openpaths/internal/model"

// Presets are ready-made agent templates surfaced in the builder gallery.
var Presets = []model.AgentPreset{
	{
		Key:         "creative",
		Name:        "Creative Studio",
		Description: "Turns an idea into a finished video: writes a scene, renders a key image, then animates it.",
		Model:       "claude-sonnet-4-6",
		SystemPrompt: "You are a creative director. Given a brief, write a vivid visual concept, " +
			"then call generate_image to render a key frame, then call generate_video with that " +
			"image_url to animate it. Always finish by returning the image and video URLs and a short caption.",
		Config: model.AgentConfig{
			Tools:      []string{"generate_image", "generate_video", "call_model"},
			MaxSteps:   8,
			ImageModel: "zimage",
			VideoModel: "seedance-2.0-image-to-video",
		},
	},
	{
		Key:         "researcher",
		Name:        "Document Researcher",
		Description: "Answers questions grounded in your connected documents and databases, with citations.",
		Model:       "claude-sonnet-4-6",
		SystemPrompt: "You are a meticulous research assistant. Use search_documents to ground every claim " +
			"in the connected sources and query_database for live data. Cite the chunks you used. " +
			"If the sources don't cover something, say so.",
		Config: model.AgentConfig{
			Tools:    []string{"search_documents", "query_database", "call_model"},
			MaxSteps: 8,
		},
	},
	{
		Key:         "operator",
		Name:        "Computer Operator",
		Description: "Drives a sandboxed desktop to complete tasks (experimental computer use).",
		Model:       "claude-sonnet-4-6",
		SystemPrompt: "You are a careful computer operator. Break the task into small steps, take a " +
			"screenshot before acting, and confirm each action's result before the next.",
		Config: model.AgentConfig{
			Tools:    []string{"computer_use", "search_documents"},
			MaxSteps: 12,
		},
	},
	{
		Key:         "generalist",
		Name:        "General Assistant",
		Description: "A blank multi-tool agent — grant it the tools and data sources you want.",
		Model:       "auto",
		SystemPrompt: "You are a capable assistant. Use the available tools when they help, and answer " +
			"concisely otherwise.",
		Config: model.AgentConfig{
			Tools:    []string{"search_documents", "generate_image", "call_model"},
			MaxSteps: 8,
		},
	},
}

// PresetByKey returns a preset template by its key.
func PresetByKey(key string) *model.AgentPreset {
	for i := range Presets {
		if Presets[i].Key == key {
			return &Presets[i]
		}
	}
	return nil
}
