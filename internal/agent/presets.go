package agent

import "github.com/openpaths/openpaths/internal/model"

// Presets are ready-made agent templates surfaced in the builder gallery.
var Presets = []model.AgentPreset{
	{
		Key:         "creative",
		Name:        "Creative Studio",
		Description: "Turns an idea into a finished video: writes a scene, renders a key image, then animates it.",
		Model:       "auto-medium-task",
		SystemPrompt: "You are a creative director. Given a brief, write a vivid visual concept, " +
			"then call generate_image to render a key frame, then call generate_video with that " +
			"image_url to animate it. Always finish by returning the image and video URLs and a short caption.",
		Config: model.AgentConfig{
			Tools:      []string{"generate_image", "generate_video", "call_model"},
			MaxSteps:   8,
			ImageModel: "zimage",
			VideoModel: "seedance-2.0-image-to-video",
		},
		ExamplePrompts: []string{
			"Create a 10-second launch film for a solar-powered backpack.",
			"Turn this product idea into a cinematic key frame and video.",
		},
	},
	{
		Key:         "researcher",
		Name:        "Document Researcher",
		Description: "Answers questions grounded in your connected documents and databases, with citations.",
		Model:       "auto-medium-task",
		SystemPrompt: "You are a meticulous research assistant. Use search_documents to ground every claim " +
			"in the connected sources and query_database for live data. Cite the chunks you used. " +
			"If the sources don't cover something, say so.",
		Config: model.AgentConfig{
			Tools:    []string{"search_documents", "query_database", "call_model"},
			MaxSteps: 8,
		},
		ExamplePrompts: []string{
			"Summarize the main decisions across these files and cite the sources.",
			"Which claims in these documents disagree with each other?",
		},
	},
	{
		Key:         "deep-researcher",
		Name:        "Deep Researcher",
		Description: "Plans multi-step web research, runs diverse searches, and synthesizes a cited answer.",
		Model:       "auto-hard-task",
		SystemPrompt: "You are a deep research agent. First lay out a short research plan for the question, " +
			"then run multiple diverse web_search queries covering different angles and phrasings. Read the " +
			"returned snippets, follow up with narrower searches where the evidence is thin, and use " +
			"search_documents for any connected sources plus call_model for heavy sub-analysis. Iterate until " +
			"you are confident, then synthesize a structured answer with clear sections and cite the source " +
			"URLs for every important claim. Note open questions or conflicting evidence explicitly.",
		Config: model.AgentConfig{
			Tools:    []string{"web_search", "search_documents", "call_model"},
			MaxSteps: 12,
		},
		ExamplePrompts: []string{
			"Research the current state of solid-state batteries and who is closest to shipping.",
			"Compare the top three vector databases for a 100M-embedding workload, with sources.",
		},
	},
	{
		Key:         "operator",
		Name:        "Computer Operator",
		Description: "Drives a sandboxed desktop to complete tasks (experimental computer use).",
		Model:       "auto-medium-task",
		SystemPrompt: "You are a careful computer operator. Break the task into small steps, take a " +
			"screenshot before acting, and confirm each action's result before the next.",
		Config: model.AgentConfig{
			Tools:    []string{"computer_use", "search_documents"},
			MaxSteps: 12,
		},
		ExamplePrompts: []string{
			"Open the report, find the quarterly total, and enter it in the form.",
			"Walk through this web workflow and stop before the final submission.",
		},
	},
	{
		Key:         "generalist",
		Name:        "General Assistant",
		Description: "A blank multi-tool agent - grant it the tools and data sources you want.",
		Model:       "auto",
		SystemPrompt: "You are a capable assistant. Use the available tools when they help, and answer " +
			"concisely otherwise.",
		Config: model.AgentConfig{
			Tools:    []string{"search_documents", "generate_image", "call_model"},
			MaxSteps: 8,
		},
		ExamplePrompts: []string{
			"Research these notes, compare options, and recommend a next step.",
			"Draft an answer using the connected files and create a supporting visual.",
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
