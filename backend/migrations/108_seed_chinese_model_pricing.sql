-- Seed Chinese model pricing data (GLM, MiniMax, Kimi)
-- Prices sourced from OpenRouter official pricing pages (USD per token)
-- All entries are locked so remote sync will not overwrite them

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

INSERT INTO model_pricings (
    model,
    input_cost_per_token,
    output_cost_per_token,
    cache_read_input_token_cost,
    cache_creation_input_token_cost,
    supports_prompt_caching,
    litellm_provider,
    mode,
    locked,
    source,
    created_at,
    updated_at
) VALUES
-- GLM-5.1: OpenRouter $0.95/M input, $3.15/M output
-- Official CNY: ¥6 input, ¥24 output, ¥1.3 cache write, ¥1.3 cache read (short ctx <32K)
('glm-5.1',
    0.95e-06,
    3.15e-06,
    0.21e-06,
    0.181e-06,
    true, 'zhipu', 'chat', true, 'manual', NOW(), NOW()),

-- GLM-5: OpenRouter $0.72/M input, $2.30/M output
-- Official CNY: ¥4 input, ¥18 output, ¥1 cache write, ¥1 cache read (short ctx <32K)
('glm-5',
    0.72e-06,
    2.30e-06,
    0.17e-06,
    0.139e-06,
    true, 'zhipu', 'chat', true, 'manual', NOW(), NOW()),

-- GLM-5-Turbo: OpenRouter $1.20/M input, $4.00/M output
-- Official CNY: ¥5 input, ¥22 output, ¥1.2 cache write, ¥1.2 cache read (short ctx <32K)
('glm-5-turbo',
    1.20e-06,
    4.00e-06,
    0.20e-06,
    0.167e-06,
    true, 'zhipu', 'chat', true, 'manual', NOW(), NOW()),

-- GLM-5V-Turbo: OpenRouter $1.20/M input, $4.00/M output
-- Official CNY: ¥5 input, ¥22 output, ¥1.2 cache write, ¥1.2 cache read (short ctx <32K)
('glm-5v-turbo',
    1.20e-06,
    4.00e-06,
    0.20e-06,
    0.167e-06,
    true, 'zhipu', 'chat', true, 'manual', NOW(), NOW()),

-- MiniMax-M2.5: OpenRouter $0.118/M input, $0.99/M output
-- Official CNY: ¥2.1 input, ¥8.4 output, ¥0.21 cache read, ¥2.625 cache write
('minimax-m2.5',
    0.118e-06,
    0.99e-06,
    0.035e-06,
    0.365e-06,
    true, 'minimax', 'chat', true, 'manual', NOW(), NOW()),

-- MiniMax-M2.7: OpenRouter $0.30/M input, $1.20/M output
-- Official CNY: ¥2.1 input, ¥8.4 output, ¥0.42 cache read, ¥2.625 cache write
('minimax-m2.7',
    0.30e-06,
    1.20e-06,
    0.058e-06,
    0.365e-06,
    true, 'minimax', 'chat', true, 'manual', NOW(), NOW()),

-- Kimi K2.5: OpenRouter $0/M input, $0/M output (free on OpenRouter)
-- Official CNY: ¥4/MTok input, ¥21/MTok output, ¥0.70/MTok cache read (using official CNY converted)
('kimi-k2.5',
    0.556e-06,
    2.917e-06,
    0.097e-06,
    NULL,
    true, 'moonshot', 'chat', true, 'manual', NOW(), NOW()),

-- Kimi K2.6: Not on OpenRouter, using Kimi K2.5 official pricing
('kimi-k2.6',
    0.556e-06,
    2.917e-06,
    0.097e-06,
    NULL,
    true, 'moonshot', 'chat', true, 'manual', NOW(), NOW()),

-- Kimi for Coding: Not on OpenRouter, using Kimi K2.5 official pricing
('kimi-for-coding',
    0.556e-06,
    2.917e-06,
    0.097e-06,
    NULL,
    true, 'moonshot', 'chat', true, 'manual', NOW(), NOW())
ON CONFLICT (model) DO UPDATE SET
    input_cost_per_token = EXCLUDED.input_cost_per_token,
    output_cost_per_token = EXCLUDED.output_cost_per_token,
    cache_read_input_token_cost = EXCLUDED.cache_read_input_token_cost,
    cache_creation_input_token_cost = EXCLUDED.cache_creation_input_token_cost,
    supports_prompt_caching = EXCLUDED.supports_prompt_caching,
    litellm_provider = EXCLUDED.litellm_provider,
    mode = EXCLUDED.mode,
    locked = EXCLUDED.locked,
    source = EXCLUDED.source,
    updated_at = NOW();
