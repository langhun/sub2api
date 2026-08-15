# Help documentation module

This Overlay module owns the authenticated help center at `/docs`. It does not change the upstream router or sidebar implementation; `routes.ts`, `navigation.ts`, and `locales.ts` are aggregated by the existing custom registries.

## Add or update a topic

1. Add matching Markdown files under `content/zh/<slug>.md` and `content/en/<slug>.md`.
2. Put beginner content after `<!-- beginner -->` and protocol details after `<!-- reference -->`.
3. Add the topic metadata to `catalog.ts`: category, audience, bilingual title/summary, order, update date, client version, sources, related topics, and screenshots.
4. Put screenshots under `assets/<client>/`. Only committed PNG or WebP files are allowed.
5. Run `pnpm docs:check`, the docs Vitest suite, lint, and typecheck.
6. Open the topic on desktop and mobile. Test search, both views, code copy, every image, and the browser console.

Routes and sidebar components do not need editing for a new topic.

## Ten-step integration tutorial contract

Every client integration tutorial must keep the beginner view in this order:

1. List the prerequisites.
2. Create an API key on this site.
3. Confirm the key's group and available models.
4. Copy the site's `api_base_url`.
5. Fill every client connection field.
6. Select an available model ID.
7. Send a minimal test message.
8. Explain the response that confirms success.
9. List the first three checks for a failed request.
10. List the diagnostic information required for further support.

Keep protocol differences, advanced parameters, and raw JSON in the reference view. Each beginner step should provide the relevant screenshot, field explanation, copyable configuration, common error, and next-step link when applicable.

## Screenshot workflow

- Crop to the fields required for the step and annotate them with red boxes, arrows, and numbered markers.
- Replace every credential with `sk-demo-...`. Official generic illustrations use the public `https://api.daigua.icu` address as a DaiGua authenticity marker; private hosts are never allowed.
- Remove accounts, email addresses, cookies, tokens, private domains, character conversations, and browser profile data.
- Record the client name, generic-interface capture method, version, capture date, and topic slug in `catalog.ts`.
- Write a complete Markdown alt value and a useful image title; the title becomes the visible caption.
- Verify the image at narrow mobile width and in the click-to-zoom viewer.

The eight generic illustrations can be regenerated on Windows with:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File src/custom/modules/docs/scripts/generate-generic-assets.ps1
```

Regeneration is appropriate only for generic illustrations. Real client screenshots must be recaptured from the documented version and reviewed for secrets.

## Source and verification policy

- `https://aiapipay.com/docs` is a baseline, not copy-ready text. The 14 baseline topics are locked by tests.
- Repository behavior takes priority for gateway endpoints, API-key fields, errors, quotas, and model filtering.
- Third-party client fields and install commands must use official sources and carry a verification date.
- Do not promise storage, billing, retry, model, or protocol behavior that is not enforced by this repository or current service policy.
- Model examples are illustrative. User steps must first call `/v1/models` with the actual key.

## Validation

From `frontend/` run:

```text
pnpm docs:check
pnpm vitest run src/custom/modules/docs/__tests__
pnpm lint:check
pnpm typecheck
pnpm test:run
```

The docs check rejects missing bilingual topics, missing markers or assets, external images, old placeholder domains, and credential-like values. It intentionally allows external source links.
