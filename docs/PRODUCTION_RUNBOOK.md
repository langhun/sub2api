# Sub2API Production Runbook

This repo uses a fixed production rollout flow. Deployment credentials and target host details are managed out of band; do not commit them into scripts, docs, or examples.

## Local Build

- Build only the Linux `amd64` production binary on Windows.
- Do not do local pseudo-production cutovers.
- Keep the upload artifact in `dist/upload/sub2api-linux-amd64`.
- Use `backend/cmd/server/VERSION` as the release version source.
- Use `./build-linux.bat` on Windows or `./build-linux.sh` on Unix-like shells.

## Target Access

- Pass the target server through environment variables when using helper scripts:
  - `REMOTE_HOST`: target host, required.
  - `REMOTE_PORT`: SSH port, defaults to `22`.
  - `REMOTE_USER`: SSH user, defaults to `root`.
- Credentials are managed out-of-band and must not be committed into the repo.

## Production Layout

- Production service: `sub2api.service`
- Production binary: `/opt/sub2api/sub2api`
- Production `DATA_DIR`: `/app/data`
- Production listen port: `8808`

## Standard Test Instance

- Test service: `sub2api-test.service`
- Test binary: `/opt/sub2api-test/sub2api`
- Test `DATA_DIR`: `/opt/sub2api-test/data`
- Test port: `18808`
- Test Mihomo data dir: `/opt/sub2api-test/data/proxy-subscription-mihomo`
- Test Mihomo listener port range: `21380-21480`

## Rollout Flow

0. Before any restart, verify the runtime config already contains:
   - a fixed `totp.encryption_key`
   - HTTPS upstream URLs for internet-facing deployments
   - `security.url_allowlist.allow_private_hosts=false` unless private upstreams are explicitly required
1. Build the Linux `amd64` production binary locally.
2. Upload the exact candidate binary or package to the server.
3. Refresh the isolated `18808` test instance first.
4. Verify the test instance before touching production.
5. Promote the exact same verified binary to `/opt/sub2api/sub2api`.
6. Restart `sub2api.service`.

## Helper Scripts

- Test instance validation: `REMOTE_HOST=<host> REMOTE_PORT=<port> REMOTE_USER=<user> ./deploy-to-test.sh`
- Production promotion: `REMOTE_HOST=<host> REMOTE_PORT=<port> REMOTE_USER=<user> ./deploy-to-production.sh`
- Scripts must fail on missing host, hash mismatch, failed health checks, or `needs_setup=true`.

## Required Verification

- Binary hash matches between local artifact, test binary, and production binary.
- `-version` output is correct.
- Recent startup logs do not contain `TOTP encryption key auto-generated` or unexpected private-host allowance.
- `http://127.0.0.1:18808/health` returns OK before promotion.
- `http://127.0.0.1:18808/setup/status` returns `needs_setup=false`.
- `http://127.0.0.1:18808/api/v1/public/pricing` returns JSON successfully.
- `http://127.0.0.1:18808/api/v1/monitoring/summary` returns JSON successfully.
- `systemctl status sub2api-test` is healthy before promotion.
- After promotion, repeat the same checks against production port `8808`.
- Review recent `journalctl -u sub2api` logs after restart.

## Cleanup Rules

- Old ad-hoc test runs such as `9908` must be removed instead of reused.
- Keep `18808` as the only standard isolated production-validation instance.
- Stop and disable `sub2api-test.service` after production is confirmed healthy unless the operator explicitly asks otherwise.
- Clean local temporary artifacts after deployment and keep the workspace tidy.
