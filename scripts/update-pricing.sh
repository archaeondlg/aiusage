#!/usr/bin/env bash
# Fetch latest LiteLLM pricing and compact it for embedding.
# Run from the repository root.
set -euo pipefail

LITELLM_URL="https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"
EMBED_DIR="internal/pricing/embed"
OUTFILE="$EMBED_DIR/litellm-pricing.json"

echo "Fetching LiteLLM pricing from $LITELLM_URL..."
curl -sL --connect-timeout 30 "$LITELLM_URL" | python3 -c "
import json, sys
raw = json.loads(sys.stdin.read())
compact = {}
count = 0
for model, pricing in raw.items():
    if not isinstance(pricing, dict):
        continue
    inp = pricing.get('input_cost_per_token')
    out = pricing.get('output_cost_per_token')
    if inp is None or out is None:
        continue
    fields = {'i': inp, 'o': out}
    cc = pricing.get('cache_creation_input_token_cost')
    if cc is not None: fields['cc'] = cc
    cr = pricing.get('cache_read_input_token_cost')
    if cr is not None: fields['cr'] = cr
    ia = pricing.get('input_cost_per_token_above_200k_tokens')
    if ia is not None: fields['ia'] = ia
    oa = pricing.get('output_cost_per_token_above_200k_tokens')
    if oa is not None: fields['oa'] = oa
    cca = pricing.get('cache_creation_input_token_cost_above_200k_tokens')
    if cca is not None: fields['cca'] = cca
    cra = pricing.get('cache_read_input_token_cost_above_200k_tokens')
    if cra is not None: fields['cra'] = cra
    ctx = pricing.get('max_input_tokens')
    if ctx is not None: fields['ctx'] = ctx
    fast = pricing.get('provider_specific_entry', {}).get('fast') if isinstance(pricing.get('provider_specific_entry'), dict) else None
    if fast is not None: fields['fast'] = fast
    compact[model] = fields
    count += 1

json.dump(compact, open('$OUTFILE', 'w'), separators=(',', ':'))
print(f'Embedded {count} models to $OUTFILE')
"
