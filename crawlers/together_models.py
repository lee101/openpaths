#!/usr/bin/env python3
"""
Crawl Together AI model catalog and store metadata in PostgreSQL.

Usage:
    TOGETHER_API_KEY=... DATABASE_URL=... python3 crawlers/together_models.py

    # Or with .env file in project root:
    python3 crawlers/together_models.py
"""

import json
import os
import sys
import requests
import psycopg2
from psycopg2.extras import Json
from datetime import datetime

def load_env():
    """Load .env file from project root."""
    env_path = os.path.join(os.path.dirname(__file__), '..', '.env')
    if os.path.exists(env_path):
        with open(env_path) as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith('#') or '=' not in line:
                    continue
                key, val = line.split('=', 1)
                val = val.strip().strip('"').strip("'")
                if key not in os.environ:
                    os.environ[key] = val

def fetch_models(api_key):
    """Fetch all models from Together AI API."""
    resp = requests.get(
        'https://api.together.xyz/v1/models',
        headers={'Authorization': f'Bearer {api_key}'},
        timeout=30,
    )
    resp.raise_for_status()
    return resp.json()

def enrich_model(model):
    """Extract structured fields from raw model data."""
    pricing = model.get('pricing', {})
    model_type = model.get('type', 'unknown')

    input_modalities = ['text']
    output_modalities = ['text']
    if model_type == 'image':
        output_modalities = ['image']
    elif model_type == 'audio':
        output_modalities = ['audio']
    elif model_type == 'video':
        output_modalities = ['video']
    elif model_type == 'transcribe':
        input_modalities = ['audio']
    elif model_type == 'embedding':
        output_modalities = ['embedding']

    features = []
    config = model.get('config', {})
    if config.get('chat_template'):
        features.append('chat_template')

    return {
        'provider': 'together',
        'model_id': model['id'],
        'display_name': model.get('display_name', model['id']),
        'organization': model.get('organization', ''),
        'model_type': model_type,
        'context_length': model.get('context_length', 0) or 0,
        'parameters': '',
        'license': model.get('license') or '',
        'link': model.get('link') or '',
        'input_price': pricing.get('input', 0) or 0,
        'output_price': pricing.get('output', 0) or 0,
        'price_per_image': pricing.get('base', 0) or 0,
        'features': Json(features),
        'input_modalities': Json(input_modalities),
        'output_modalities': Json(output_modalities),
        'deployment': Json(['serverless'] if model.get('running') or True else []),
        'raw_metadata': Json(model),
    }

def upsert_model(cur, data):
    """Insert or update model metadata."""
    cur.execute("""
        INSERT INTO model_metadata (
            provider, model_id, display_name, organization, model_type,
            context_length, parameters, license, link,
            input_price, output_price, price_per_image,
            features, input_modalities, output_modalities,
            deployment, raw_metadata, last_scraped_at, updated_at
        ) VALUES (
            %(provider)s, %(model_id)s, %(display_name)s, %(organization)s, %(model_type)s,
            %(context_length)s, %(parameters)s, %(license)s, %(link)s,
            %(input_price)s, %(output_price)s, %(price_per_image)s,
            %(features)s, %(input_modalities)s, %(output_modalities)s,
            %(deployment)s, %(raw_metadata)s, now(), now()
        )
        ON CONFLICT (provider, model_id) DO UPDATE SET
            display_name = EXCLUDED.display_name,
            organization = EXCLUDED.organization,
            model_type = EXCLUDED.model_type,
            context_length = EXCLUDED.context_length,
            license = EXCLUDED.license,
            link = EXCLUDED.link,
            input_price = EXCLUDED.input_price,
            output_price = EXCLUDED.output_price,
            price_per_image = EXCLUDED.price_per_image,
            features = EXCLUDED.features,
            input_modalities = EXCLUDED.input_modalities,
            output_modalities = EXCLUDED.output_modalities,
            deployment = EXCLUDED.deployment,
            raw_metadata = EXCLUDED.raw_metadata,
            last_scraped_at = now(),
            updated_at = now()
    """, data)

def main():
    load_env()

    api_key = os.environ.get('TOGETHER_API_KEY')
    if not api_key:
        print('TOGETHER_API_KEY not set')
        sys.exit(1)

    db_url = os.environ.get('DATABASE_URL', 'postgres://openpath:openpath@localhost:5432/openpath?sslmode=disable')
    json_only = '--json' in sys.argv

    print(f'Fetching models from Together AI...', file=sys.stderr)
    models = fetch_models(api_key)
    print(f'Found {len(models)} models', file=sys.stderr)

    if json_only:
        enriched = [enrich_model(m) for m in models]
        for e in enriched:
            for k, v in e.items():
                if isinstance(v, Json):
                    e[k] = v.adapted
        json.dump(enriched, sys.stdout, indent=2, default=str)
        print()
        return

    conn = psycopg2.connect(db_url)
    cur = conn.cursor()

    # Ensure table exists (mirrors 003_model_metadata.sql)
    cur.execute("""
        CREATE TABLE IF NOT EXISTS model_metadata (
            id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            provider        TEXT NOT NULL,
            model_id        TEXT NOT NULL,
            display_name    TEXT NOT NULL DEFAULT '',
            organization    TEXT NOT NULL DEFAULT '',
            model_type      TEXT NOT NULL DEFAULT 'chat',
            context_length  INT NOT NULL DEFAULT 0,
            parameters      TEXT NOT NULL DEFAULT '',
            license         TEXT NOT NULL DEFAULT '',
            link            TEXT NOT NULL DEFAULT '',
            input_price     NUMERIC(12,6) NOT NULL DEFAULT 0,
            output_price    NUMERIC(12,6) NOT NULL DEFAULT 0,
            price_per_image NUMERIC(12,6) NOT NULL DEFAULT 0,
            features        JSONB NOT NULL DEFAULT '[]',
            input_modalities  JSONB NOT NULL DEFAULT '["text"]',
            output_modalities JSONB NOT NULL DEFAULT '["text"]',
            deployment      JSONB NOT NULL DEFAULT '[]',
            raw_metadata    JSONB NOT NULL DEFAULT '{}',
            last_scraped_at TIMESTAMPTZ NOT NULL DEFAULT now(),
            created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
            updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
            UNIQUE(provider, model_id)
        )
    """)
    conn.commit()

    counts = {}
    for model in models:
        data = enrich_model(model)
        upsert_model(cur, data)
        t = data['model_type']
        counts[t] = counts.get(t, 0) + 1

    conn.commit()
    cur.close()
    conn.close()

    print(f'Upserted {len(models)} models:')
    for t, c in sorted(counts.items()):
        print(f'  {t}: {c}')

if __name__ == '__main__':
    main()
