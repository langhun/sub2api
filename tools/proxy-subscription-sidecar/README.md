# Proxy Subscription Sidecar

Minimal companion service for Sub2API proxy subscription runtimes.

Current scope:
- health check endpoint
- runtime upsert/delete/check endpoints
- in-memory listener port allocation
- sing-box config generation
- sing-box process spawn / status tracking

Current limitation:
- protocol mapping is still minimal and focuses on common fields
- advanced transport / reality / plugin options are not fully mapped yet

Notes:
- If `SING_BOX_BIN` is not provided, the sidecar resolves `sing-box` using `where sing-box`
- In Docker deployments, the bundled image installs `sing-box` directly
