# Overlay boundary gate

`check-custom-overlay-boundaries.ps1` has three explicit audit modes. No mode
accepts a floating `origin/main` reference: upstream synchronization can
legitimately change shared files, so every review records an immutable SHA.

`IncrementalOverlay` is the default. It compares the target to the reviewed
post-sync product checkpoint `c27d1237628cac3e41a8586c3fb5c9dcbd2bc19c`.
Use it for the current migration: identity, admin, and other shared-path debt
that existed before this checkpoint is inventory, not a new regression.

`HistoricalAudit` uses the named product freeze
`baseline/v0.1.164-before-overlay` to produce that full inventory. It is not an
acceptance gate for one incremental module move. `OverlayBoundary` compares a
product target with a reviewed upstream SHA which must be its ancestor. Use it
only for an actual upstream merge and pair it with `check-upstream-sync.ps1`.

```powershell
# Audit the current incremental migration from c27d12376 and include worktree
# changes. This is the normal pre-commit command.
powershell -NoProfile -ExecutionPolicy Bypass -File tools/check-custom-overlay-boundaries.ps1 `
  -IncludeWorktree

# Audit the historical downstream inventory without changing the incremental
# acceptance baseline.
powershell -NoProfile -ExecutionPolicy Bypass -File tools/check-custom-overlay-boundaries.ps1 `
  -Mode HistoricalAudit `
  -IncludeWorktree
```

```powershell
# Normal module/upstream synchronization boundary check. Record the exact SHA
# in the sync commit or CI metadata; do not substitute origin/main here.
powershell -NoProfile -ExecutionPolicy Bypass -File tools/check-custom-overlay-boundaries.ps1 `
  -Mode OverlayBoundary `
  -UpstreamRef 43d4bae24 `
  -TargetRef HEAD `
  -IncludeWorktree `
  -ReportPath artifacts/overlay-boundary.json
```

For an upstream merge, create a pre-sync and post-sync report with the exact
old/new upstream commits, then compare them. The second gate rejects a merge
that introduces new shared paths while reporting upstream paths separately.

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File tools/check-upstream-sync.ps1 `
  -ProductBefore <pre-sync-product-sha> `
  -ProductAfter HEAD `
  -OldUpstream <old-upstream-sha> `
  -NewUpstream 43d4bae24 `
  -PreReport artifacts/overlay-pre.json `
  -PostReport artifacts/overlay-post.json
```

The gate reports the following ownership categories separately:

| Category | Allowed content |
| --- | --- |
| `custom-backend` / `custom-frontend` | Downstream business modules and their private implementation. |
| `fixed-mount` | One registration, aggregation, route, settings, navigation, or locale mount at an approved host seam. |
| `composition-root` | Application construction only: Wire invokes `custom` providers but owns no module rules. |
| `restricted-platform-port` | A named, generic host interface used by an Overlay, with no import of a custom business module. |
| `verification-test` | Tests only; production behavior is still checked by the other categories. |
| `published-immutable-migration` | A known migration that predates the chosen baseline. |
| `historical-upstream-revert` | A shared path whose final content exactly matches the immutable upstream SHA for the current incremental baseline. |

Known migrations `173` through `181`, `185`, and `187` are immutable. Adding
them from an older baseline is reported as historical data; modifying, deleting,
or renaming one is an `immutable-migration-change` violation. A shared path is
only cleanup when its diff has zero added lines. There is no path-based cleanup
allowlist, so a future addition cannot be hidden behind a historical deletion.
The sole mechanical exception is the reviewed removal of the unused
`RegisterCommonRoutes` handler argument: the gate accepts only its one exact
replacement signature and rejects every other added line in that path.
For the current incremental baseline only, a shared path may also be classified
as `historical-upstream-revert` when it matches upstream commit `43d4bae24`
byte-for-byte. This proves that the migration returned the file to upstream;
it is not a path allowlist and does not apply to actual upstream sync audits.

Any new production business path outside these categories fails the gate. A
new mount, composition root, or platform port needs a separate reviewed
allowlist change that documents why the generic host seam is necessary.

When no commits exist after the selected baseline, the script prints that the
committed audit is not applicable rather than claiming a full pass. The
worktree is excluded unless `-IncludeWorktree` is supplied.
