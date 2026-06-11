# Sub2API Production Runbook

This repo uses a fixed production rollout flow. Deployment credentials and target host details are managed out of band; do not commit them into scripts, docs, or examples.

## Local Build

- Build only the Linux `amd64` production binary on Windows.
- Do not do local pseudo-production cutovers.
- Keep the upload artifact in `dist/upload/sub2api-linux-amd64`.
- Keep the upload package stable as `dist/upload/sub2api-linux-amd64.tar.zst` and fall back to `dist/upload/sub2api-linux-amd64.tar.gz` when `zstd` is unavailable.
- Use `backend/cmd/server/VERSION` as the release version source.
- Use `./build-linux.bat` on Windows or `./build-linux.sh` on Unix-like shells.
- The Windows build script falls back to `frontend/node_modules/.bin/vite.cmd` when `pnpm` / `corepack` are not on `PATH`.

## Target Access

- Pass the target server through environment variables when using helper scripts:
  - `REMOTE_HOST`: target host, required.
  - `REMOTE_PORT`: SSH port, defaults to `22`.
  - `REMOTE_USER`: SSH user, defaults to `root`.
- `REMOTE_UPLOAD_DIR`: remote upload/staging directory, defaults to `/opt/sub2api-rollout`.
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
   - if the config still carries legacy `security.url_allowlist` fields, keep `enabled=true` and `allow_private_hosts=false` unless private upstreams are explicitly required
1. Build the Linux `amd64` production binary locally.
2. Upload the exact candidate package and `deploy/remote-production-flow.sh` to the remote staging directory.
3. Run the remote flow script in `test` or `full` mode.
4. Refresh the isolated `18808` test instance first.
5. Verify the test instance before touching production.
6. Promote the exact same verified binary to `/opt/sub2api/sub2api`.
7. Restart `sub2api.service`, verify `8808`, then stop/disable `sub2api-test.service` unless explicitly kept.

## Helper Scripts

- Windows one-shot rollout: `.\production-rollout.ps1 -RemoteHost <host> -RemotePort <port> -RemoteUser <user> -RemoteUploadDir <dir>`
- Unix test validation wrapper: `REMOTE_HOST=<host> REMOTE_PORT=<port> REMOTE_USER=<user> REMOTE_UPLOAD_DIR=<dir> ./deploy-to-test.sh`
- Unix production promotion wrapper: `REMOTE_HOST=<host> REMOTE_PORT=<port> REMOTE_USER=<user> REMOTE_UPLOAD_DIR=<dir> ./deploy-to-production.sh`
- Remote server flow entry: `deploy/remote-production-flow.sh` with modes `test`, `promote`, `full`
- Scripts must fail on missing host, missing artifacts, hash mismatch, failed health checks, or `needs_setup=true`.

## Required Verification

- Binary hash matches between local artifact, test binary, and production binary.
- `-version` output is correct.
- Config precheck must reject empty `totp.encryption_key`.
- If legacy `security.url_allowlist` keys still exist in the runtime config, `enabled=true` and `allow_private_hosts=false` must be enforced before rollout.
- Recent startup logs do not contain `TOTP encryption key auto-generated`.
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
