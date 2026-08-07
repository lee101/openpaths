package model

// ArtImage is a single record in the image prompt search engine (art_images table).
// JSON tags match the public /v1/art API shape (and the frontend ZImageArtItem type).
type ArtImage struct {
	ID        string   `json:"id"`
	Slug      string   `json:"slug"`
	Prompt    string   `json:"prompt"`
	Title     string   `json:"title,omitempty"`
	Width     int      `json:"width,omitempty"`
	Height    int      `json:"height,omitempty"`
	Aspect    string   `json:"aspect,omitempty"`
	ImageURL  string   `json:"imageUrl"`
	ThumbURL  string   `json:"thumbUrl,omitempty"`
	Model     string   `json:"model"`
	Seed      int64    `json:"seed,omitempty"`
	Steps     int      `json:"steps,omitempty"`
	Source    string   `json:"source,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	NSFW      bool     `json:"nsfw,omitempty"`
	CreatedAt string   `json:"createdAt,omitempty"`
	// Video gallery fields. MediaType is "image" (default) or "video".
	MediaType       string `json:"mediaType,omitempty"`
	VideoURL        string `json:"videoUrl,omitempty"`
	PosterURL       string `json:"posterUrl,omitempty"`
	DurationSeconds int    `json:"durationSeconds,omitempty"`
}

// ArtImageFilters narrows browse/search queries.
type ArtImageFilters struct {
	Aspect    string // "square" | "portrait" | "wide" | "" (any)
	Tag       string // single tag match
	Source    string // "cutedsl" | "openpaths-gen" | "" (any)
	MediaType string // "image" | "video" | "" (any)
	MinW      int
	MinH      int
}

// AspectFor classifies width/height into the aspect bucket used for filtering.
func AspectFor(width, height int) string {
	switch {
	case width <= 0 || height <= 0 || width == height:
		return "square"
	case height > width:
		return "portrait"
	default:
		return "wide"
	}
}
