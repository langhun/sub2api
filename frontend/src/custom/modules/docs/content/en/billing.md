<!-- beginner -->
## Step 1: Separate balance, subscriptions, and key limits

| Item | Purpose | When exhausted |
| --- | --- | --- |
| Account balance | Pays actual cost for balance groups | Requests are rejected for insufficient balance |
| Subscription quota | Usage budget for a subscription group and validity period | The subscription cannot be used after quota or expiry |
| Key total quota | Cumulative limit for one key | That key is rejected; other keys are unaffected |
| 5-hour/day/7-day limits | Rolling or periodic budget for one key | Requests fail until the window recovers |

Balance and subscriptions are separate billing sources and are not automatically converted. The key's group determines the billing mode; do not assume an exhausted subscription automatically falls back to balance.

## Step 2: Understand multipliers

**Original cost** in a usage record is calculated before the effective multiplier. **Billed cost** is the amount after the current multiplier. Balance, subscription quota, key quota, and periodic limits accumulate the billed cost.

The effective multiplier may come from:

- the group's default multiplier;
- a user-specific multiplier set by an administrator;
- a peak-period multiplier;
- an independent multiplier for images or other media.

Do not infer every request from one group number. Use the model catalog, key page, and the multiplier plus billed cost on the specific usage record.

## Step 3: Reconcile one request

After a minimal request, refresh **Usage** and compare timestamp, key, requested model, upstream model, protocol, input/output tokens, cache tokens, multiplier, original cost, billed cost, and status. For interrupted streams, tools, and media, use the final usage record.

## Step 4: Identify quota errors

An exhausted key quota commonly returns `429 API_KEY_QUOTA_EXHAUSTED` or OpenAI-style `insufficient_quota`. Insufficient account balance commonly returns `403 INSUFFICIENT_BALANCE`. Ordinary rate limiting can also return `429`; use the error code and `Retry-After`, not only the status code.

## Frequently asked questions

### Should I use a subscription or balance?

Subscriptions fit a fixed group and periodic budget. Balance fits actual-cost billing across available balance groups. Before purchase, check validity, periodic quota, supported groups, concurrency, and multiplier. Existing subscriptions and balance are not automatically merged by a new purchase.

### Are custom enterprise packages available?

Use **Contact support** at the top of the article and include required models, expected monthly usage, concurrency, protocols, and settlement needs. Availability, pricing, quota, and delivery time require an administrator-confirmed proposal; this guide makes no fixed promise.

### Why is there no refund button?

The order page shows the action only when the payment provider enables both refunds and user refund requests and the order is eligible. If it is absent, use **Contact support** with the order number instead of creating duplicate orders.

### How long does a refund take?

A refund passes order validation, administrator processing, and payment-provider confirmation. It may be requested, refunding, partially refunded, refunded, or failed. There is no fixed business-day promise. The refundable amount can also depend on prior refunds, recoverable balance, subscription benefits, and promotional credit.

<!-- reference -->
## Billing semantics

The gateway records pre-multiplier `TotalCost` and post-multiplier `ActualCost`. Balance, subscription, and user-facing key limits use `ActualCost`; upstream account quota may use a separate account multiplier. Reconcile disputes with individual usage records and request IDs rather than dividing an upstream invoice by request count.

Refunds are available only for supported orders and payment-provider instances that currently allow them. A user request enters the refund workflow instead of directly changing the order or balance; use the final order status and refund breakdown as the result.
