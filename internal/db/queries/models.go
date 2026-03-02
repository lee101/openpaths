package queries

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ModelMetadata struct {
	ID               string          `json:"id"`
	Provider         string          `json:"provider"`
	ModelID          string          `json:"model_id"`
	DisplayName      string          `json:"display_name"`
	Organization     string          `json:"organization"`
	ModelType        string          `json:"model_type"`
	ContextLength    int             `json:"context_length"`
	Parameters       string          `json:"parameters"`
	License          string          `json:"license"`
	Link             string          `json:"link"`
	InputPrice       float64         `json:"input_price"`
	OutputPrice      float64         `json:"output_price"`
	PricePerImage    float64         `json:"price_per_image"`
	Features         json.RawMessage `json:"features"`
	InputModalities  json.RawMessage `json:"input_modalities"`
	OutputModalities json.RawMessage `json:"output_modalities"`
	Deployment       json.RawMessage `json:"deployment"`
	RawMetadata      json.RawMessage `json:"raw_metadata"`
	LastScrapedAt    time.Time       `json:"last_scraped_at"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type ModelMetadataQueries struct {
	pool *pgxpool.Pool
}

func NewModelMetadataQueries(pool *pgxpool.Pool) *ModelMetadataQueries {
	return &ModelMetadataQueries{pool: pool}
}

func defaultJSON(v json.RawMessage, fallback string) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage(fallback)
	}
	return v
}

func (q *ModelMetadataQueries) Upsert(ctx context.Context, m *ModelMetadata) error {
	features := defaultJSON(m.Features, "[]")
	inputMod := defaultJSON(m.InputModalities, `["text"]`)
	outputMod := defaultJSON(m.OutputModalities, `["text"]`)
	deployment := defaultJSON(m.Deployment, "[]")
	rawMeta := defaultJSON(m.RawMetadata, "{}")

	_, err := q.pool.Exec(ctx, `
		INSERT INTO model_metadata (provider, model_id, display_name, organization, model_type,
			context_length, parameters, license, link, input_price, output_price, price_per_image,
			features, input_modalities, output_modalities, deployment, raw_metadata, last_scraped_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,now(),now())
		ON CONFLICT (provider, model_id) DO UPDATE SET
			display_name=EXCLUDED.display_name, organization=EXCLUDED.organization,
			model_type=EXCLUDED.model_type, context_length=EXCLUDED.context_length,
			parameters=EXCLUDED.parameters, license=EXCLUDED.license, link=EXCLUDED.link,
			input_price=EXCLUDED.input_price, output_price=EXCLUDED.output_price,
			price_per_image=EXCLUDED.price_per_image, features=EXCLUDED.features,
			input_modalities=EXCLUDED.input_modalities, output_modalities=EXCLUDED.output_modalities,
			deployment=EXCLUDED.deployment, raw_metadata=EXCLUDED.raw_metadata,
			last_scraped_at=now(), updated_at=now()`,
		m.Provider, m.ModelID, m.DisplayName, m.Organization, m.ModelType,
		m.ContextLength, m.Parameters, m.License, m.Link, m.InputPrice, m.OutputPrice,
		m.PricePerImage, features, inputMod, outputMod, deployment, rawMeta)
	return err
}

func (q *ModelMetadataQueries) ListByProvider(ctx context.Context, provider string) ([]*ModelMetadata, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT provider, model_id, display_name, organization, model_type, context_length,
			features, input_modalities, output_modalities, input_price, output_price, raw_metadata,
			last_scraped_at FROM model_metadata WHERE provider=$1 ORDER BY model_id`, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*ModelMetadata
	for rows.Next() {
		m := &ModelMetadata{}
		if err := rows.Scan(&m.Provider, &m.ModelID, &m.DisplayName, &m.Organization,
			&m.ModelType, &m.ContextLength, &m.Features, &m.InputModalities,
			&m.OutputModalities, &m.InputPrice, &m.OutputPrice, &m.RawMetadata,
			&m.LastScrapedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

func (q *ModelMetadataQueries) ListAll(ctx context.Context) ([]*ModelMetadata, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT provider, model_id, display_name, organization, model_type, context_length,
			features, input_modalities, output_modalities, input_price, output_price, raw_metadata,
			last_scraped_at FROM model_metadata ORDER BY provider, model_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*ModelMetadata
	for rows.Next() {
		m := &ModelMetadata{}
		if err := rows.Scan(&m.Provider, &m.ModelID, &m.DisplayName, &m.Organization,
			&m.ModelType, &m.ContextLength, &m.Features, &m.InputModalities,
			&m.OutputModalities, &m.InputPrice, &m.OutputPrice, &m.RawMetadata,
			&m.LastScrapedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}
