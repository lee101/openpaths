#!/usr/bin/env python3
import json, sys, yaml

cfg = yaml.safe_load(open('config.yaml'))
out = []
for m in cfg['models']:
    if not m.get('input_price_per_1m') and not m.get('output_price_per_1m'):
        continue
    if any(m.get(k) for k in ('price_per_image', 'price_per_request', 'price_per_megapixel', 'price_per_1m_characters', 'price_per_second', 'price_per_image_by_resolution')):
        continue
    e = {
        'id': m['id'], 'provider': m['provider'], 'upstream': m.get('provider_model_id'),
        'input': m.get('input_price_per_1m', 0), 'output': m.get('output_price_per_1m', 0),
        'context': m.get('context_window', 0), 'maxOutput': m.get('max_output_tokens', 0),
        'vision': bool(m.get('supports_vision')), 'tools': bool(m.get('supports_tools')),
        'aliases': m.get('aliases', []),
    }
    if m.get('input_cache_hit_price_per_1m'):
        e['cacheRead'] = m['input_cache_hit_price_per_1m']
    if m.get('long_context_threshold'):
        e['longContext'] = {
            'threshold': m['long_context_threshold'],
            'input': m.get('input_price_per_1m_long') or e['input'],
            'output': m.get('output_price_per_1m_long') or e['output'],
            'cacheRead': m.get('input_cache_hit_price_per_1m_long') or e.get('cacheRead'),
        }
    s = m.get('scheduled_token_pricing')
    if s:
        e['schedule'] = {
            'effectiveAt': str(s['effective_at']),
            'windowsUtc': [[w['start'], w['end']] for w in s.get('peak_windows_utc', [])],
            'peak': {'input': s.get('peak_input_price_per_1m', e['input']), 'output': s.get('peak_output_price_per_1m', e['output']), 'cacheRead': s.get('peak_input_cache_hit_price_per_1m', e.get('cacheRead'))},
            'offPeak': {'input': s.get('off_peak_input_price_per_1m', e['input']), 'output': s.get('off_peak_output_price_per_1m', e['output']), 'cacheRead': s.get('off_peak_input_cache_hit_price_per_1m', e.get('cacheRead'))},
            'weekendOffPeak': bool(s.get('weekend_off_peak_effective_at')),
        }
    out.append(e)

json.dump(out, open('src/data/pricingConfig.generated.json', 'w'), indent=1)
print(f'wrote {len(out)} priced models', file=sys.stderr)
