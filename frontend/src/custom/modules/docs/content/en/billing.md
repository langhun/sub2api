<!-- beginner -->
## Step 1: Distinguish four limits

| Item | Purpose |
| --- | --- |
| Account balance | Spendable balance shared by all keys |
| Subscription quota | Allowance within a subscription and time window |
| Key total quota | Cumulative cap for one key |
| 5-hour/day/7-day limits | Rolling or periodic limits for one key |

## Step 2: Inspect usage

After a minimal request, open **Usage** and refresh manually. Check time, requested model, upstream model, input/output tokens, multiplier, cost, and status. The page does not promise an automatic 15-second refresh.

## Step 3: Identify errors

Exhausted key quota normally returns `429 API_KEY_QUOTA_EXHAUSTED` or OpenAI-style `insufficient_quota`; insufficient account balance is normally `403 INSUFFICIENT_BALANCE`. A `429` rate limit does not always mean the balance is empty.

<!-- reference -->
## Billing boundaries

Pricing depends on model, group multiplier, tokens, and media usage. Do not assume every 4xx/5xx is uncharged or an interrupted stream is billed only for visible text. Use the site's usage record and current policy. For disputes provide time, model, amount, and `X-Request-ID`, never the key.
