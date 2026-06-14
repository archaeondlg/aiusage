#!/usr/bin/env bash
# Compare model_pricing.csv with config.json pricing, update if changed.
# Run from the repository root.
set -euo pipefail

CSV_FILE="model_pricing.csv"
CONFIG_FILE="config.json"

if [ ! -f "$CSV_FILE" ]; then
    echo "ERROR: $CSV_FILE not found in current directory"
    exit 1
fi

if [ ! -f "$CONFIG_FILE" ]; then
    echo "→ $CONFIG_FILE not found, generating new one..."
    cp /dev/null "$CONFIG_FILE"
fi

python3 -c "
import csv, json, sys

# Parse CSV into pricing map
csv_pricing = {}
with open('$CSV_FILE') as f:
    for row in csv.DictReader(f):
        display = row['display_name'].strip()
        if not display:
            continue
        inp = float(row['input_cost_per_million']) / 1_000_000
        out = float(row['output_cost_per_million']) / 1_000_000
        entry = {'input': inp, 'output': out}
        cr = float(row.get('cache_read_cost_per_million', 0) or 0)
        if cr > 0:
            entry['cacheRead'] = cr / 1_000_000
        cc = float(row.get('cache_creation_cost_per_million', 0) or 0)
        if cc > 0:
            entry['cacheCreate'] = cc / 1_000_000
        # CSV may have duplicate display names; keep first
        if display not in csv_pricing:
            csv_pricing[display] = entry

# Load existing config
try:
    with open('$CONFIG_FILE') as f:
        config = json.load(f)
except (json.JSONDecodeError, FileNotFoundError):
    config = {}

old_pricing = config.get('pricing', {})

# Convert display name -> model key using same mapping as gen_config.go
def model_key(name):
    m = {
        'Claude Opus 4.8': 'claude-opus-4-8',
        'Claude Opus 4.7': 'claude-opus-4-7',
        'Claude Opus 4.6': 'claude-opus-4-6',
        'Claude Sonnet 4.6': 'claude-sonnet-4-6',
        'Claude Opus 4.5': 'claude-opus-4-5',
        'Claude Sonnet 4.5': 'claude-sonnet-4-5',
        'Claude Haiku 4.5': 'claude-haiku-4-5',
        'Claude Opus 4': 'claude-opus-4',
        'Claude Opus 4.1': 'claude-opus-4-1',
        'Claude Sonnet 4': 'claude-sonnet-4',
        'Claude 3.5 Haiku': 'claude-3-5-haiku',
        'Claude 3.5 Sonnet': 'claude-3-5-sonnet',
        'GPT-5.5': 'gpt-5.5',
        'GPT-5.4': 'gpt-5.4',
        'GPT-5.4 Mini': 'gpt-5.4-mini',
        'GPT-5.4 Nano': 'gpt-5.4-nano',
        'GPT-5.3 Codex': 'gpt-5.3-codex',
        'GPT-5.2': 'gpt-5.2',
        'GPT-5.2 Codex': 'gpt-5.2-codex',
        'GPT-5.1': 'gpt-5.1',
        'GPT-5.1 Codex': 'gpt-5.1-codex',
        'GPT-5': 'gpt-5',
        'GPT-5 Codex': 'gpt-5-codex',
        'GPT-5 Mini': 'gpt-5-mini',
        'GPT-5 Nano': 'gpt-5-nano',
        'GPT-4.1': 'gpt-4.1',
        'GPT-4.1 Mini': 'gpt-4.1-mini',
        'GPT-4.1 Nano': 'gpt-4.1-nano',
        'OpenAI o3-pro': 'openai-o3-pro',
        'OpenAI o3': 'openai-o3',
        'OpenAI o4-mini': 'openai-o4-mini',
        'OpenAI o3-mini': 'openai-o3-mini',
        'OpenAI o1': 'openai-o1',
        'OpenAI o1-mini': 'openai-o1-mini',
        'Gemini 3.5 Flash': 'gemini-3.5-flash',
        'Gemini 3.1 Pro Preview': 'gemini-3.1-pro',
        'Gemini 3.1 Flash Lite Preview': 'gemini-3.1-flash-lite',
        'Gemini 3.1 Flash Lite': 'gemini-3.1-flash-lite',
        'Gemini 3 Pro Preview': 'gemini-3-pro',
        'Gemini 3 Flash Preview': 'gemini-3-flash',
        'Gemini 2.5 Pro': 'gemini-2.5-pro',
        'Gemini 2.5 Flash': 'gemini-2.5-flash',
        'Gemini 2.5 Flash Lite': 'gemini-2.5-flash-lite',
        'Gemini 2.0 Flash': 'gemini-2.0-flash',
        'DeepSeek V4 Pro': 'deepseek-v4-pro',
        'DeepSeek V4 Flash': 'deepseek-v4-flash',
        'DeepSeek V3.2': 'deepseek-v3.2',
        'DeepSeek V3.1': 'deepseek-v3.1',
        'DeepSeek V3': 'deepseek-v3',
        'DeepSeek Chat': 'deepseek-chat',
        'DeepSeek Reasoner': 'deepseek-reasoner',
        'Qwen3.6 Plus': 'qwen3.6-plus',
        'Qwen3.5 Plus': 'qwen3.5-plus',
        'Qwen3 Max': 'qwen3-max',
        'Qwen3 235B-A22B': 'qwen3-235b-a22b',
        'Qwen3 Coder Plus': 'qwen3-coder-plus',
        'Qwen3 Coder Flash': 'qwen3-coder-flash',
        'Qwen3 Coder Next': 'qwen3-coder-next',
        'Qwen3 Coder 480B': 'qwen3-coder-480b',
        'Qwen3 Coder 480B-A35B Instruct': 'qwen3-coder-480b-a35b-instruct',
        'Qwen3 32B': 'qwen3-32b',
        'QwQ Plus': 'qwq-plus',
        'QwQ 32B': 'qwq-32b',
        'Kimi K2 Thinking': 'kimi-k2-thinking',
        'Kimi K2': 'kimi-k2',
        'Kimi K2 Turbo': 'kimi-k2-turbo',
        'Kimi K2.5': 'kimi-k2.5',
        'Kimi K2.6': 'kimi-k2.6',
        'GLM-4.7': 'glm-4.7',
        'GLM-4.6': 'glm-4.6',
        'GLM-5': 'glm-5',
        'GLM-5.1': 'glm-5.1',
        'Grok 4.20 Reasoning': 'grok-4.20',
        'Grok 4.20': 'grok-4.20',
        'Grok 4.1 Fast Reasoning': 'grok-4.1-fast',
        'Grok 4.1 Fast': 'grok-4.1-fast',
        'Grok 4': 'grok-4',
        'Grok Build 0.1': 'grok-build-0.1',
        'Grok Build 0.1 (Code Fast Alias)': 'grok-build-0.1-code',
        'Grok 3': 'grok-3',
        'Grok 3 Mini': 'grok-3-mini',
        'Codestral': 'codestral',
        'Devstral Small 1.1': 'devstral-small-1.1',
        'Devstral 2': 'devstral-2',
        'Devstral Medium': 'devstral-medium',
        'Mistral Large 3': 'mistral-large-3',
        'Mistral Medium 3.1': 'mistral-medium-3.1',
        'Mistral Small 3.2': 'mistral-small-3.2',
        'Magistral Medium': 'magistral-medium',
        'Cohere Command A': 'cohere-command-a',
        'Cohere Command R+': 'cohere-command-r-plus',
        'Cohere Command R': 'cohere-command-r',
        'Step 3.5 Flash': 'step-3.5-flash',
        'Step 3.5 Flash 2603': 'step-3.5-flash-2603',
        'MiniMax M2.7': 'minimax-m2.7',
        'MiniMax M2.7 Highspeed': 'minimax-m2.7-highspeed',
        'MiniMax M2.5': 'minimax-m2.5',
        'MiniMax M2.5 Lightning': 'minimax-m2.5-lightning',
        'MiniMax M2.1': 'minimax-m2.1',
        'MiniMax M2.1 Lightning': 'minimax-m2.1-lightning',
        'MiniMax M2': 'minimax-m2',
        'MiMo V2 Flash': 'mimo-v2-flash',
        'MiMo V2 Pro': 'mimo-v2-pro',
        'MiMo V2.5': 'mimo-v2.5',
        'MiMo V2.5 Pro': 'mimo-v2.5-pro',
        'Doubao Seed Code': 'doubao-seed-code',
        'Doubao Seed 2.0 Pro': 'doubao-seed-2.0-pro',
        'Doubao Seed 2.0 Code': 'doubao-seed-2.0-code',
        'Doubao Seed 2.0 Code Preview': 'doubao-seed-2.0-code-preview',
        'Doubao Seed 2.0 Lite': 'doubao-seed-2.0-lite',
        'Doubao Seed 2.0 Mini': 'doubao-seed-2.0-mini',
        'Codex Mini': 'codex-mini',
    }
    return m.get(name, name.lower().replace(' ', '-').replace('.', '-'))

# Build new pricing map from CSV
new_pricing = {}
for display, entry in csv_pricing.items():
    key = model_key(display)
    new_pricing[key] = entry

# Compare
added = []
updated = []
removed = []

for key, entry in new_pricing.items():
    if key not in old_pricing:
        added.append(key)
    elif old_pricing[key] != entry:
        updated.append(key)

for key in old_pricing:
    if key not in new_pricing:
        removed.append(key)

# Print diff
if not added and not updated and not removed:
    print('→ 价格无变化')
    sys.exit(0)

if added:
    print(f'  + 新增 {len(added)}: {\", \".join(sorted(added)[:10])}{\"...\" if len(added) > 10 else \"\"}')
if updated:
    preview = sorted(updated)[:10]
    print(f'  ~ 更新 {len(updated)}: {\", \".join(preview)}{\"...\" if len(updated) > 10 else \"\"}')
if removed:
    print(f'  - 移除 {len(removed)}: {\", \".join(sorted(removed)[:10])}{\"...\" if len(removed) > 10 else \"\"}')

# Update config
config['pricing'] = new_pricing
with open('$CONFIG_FILE', 'w') as f:
    json.dump(config, f, indent=2, ensure_ascii=False)
    f.write('\n')

print(f'  ✓ config.json 已更新 ({len(new_pricing)} 个模型)')
"
