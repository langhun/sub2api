# Overlay boundary gate

`check-custom-overlay-boundaries.ps1` has two explicit audit modes. Neither
mode reads a floating `origin/main` reference: upstream synchronization can
legitimately change shared files, so each CI run or merge review must record an
immutable commit SHA.

`HistoricalAudit` is the default and uses the named product freeze
`baseline/v0.1.164-before-overlay`. Use it for a one-off full historical audit.
`OverlayBoundary` compares `HEAD` with a reviewed upstream SHA which must be a
target ancestor. It is the normal gate before accepting an upstream sync or a
new module change; it excludes the upstream changes already merged into the
product branch.

```powershell
# Audit committed changes from the product freeze to HEAD.
powershell -NoProfile -ExecutionPolicy Bypass -File tools/check-custom-overlay-boundaries.ps1

# Audit a single migration checkpoint and include staged, unstaged, and
# untracked files. Run this before committing an in-progress migration.
powershell -NoProfile -ExecutionPolicy Bypass -File tools/check-custom-overlay-boundaries.ps1 `
  -BaselineRef 4d73b9d8de8e88939ee9824b9d540802574f0240 `
  -TargetRef HEAD `
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

The gate allows code under `backend/internal/custom/**`,
`frontend/src/custom/**`, generated Ent files, custom-named migrations,
documentation, reviewed fixed integration points, and an exact published-history
set. The reviewed cleanup set exists only for files whose net change removes old
Overlay behavior but whose formatting makes Git report replacement lines. Any new
business path outside these categories fails the gate.

When no commits exist after the selected baseline, the script prints that the
committed audit is not applicable rather than claiming a full pass. The
worktree is excluded unless `-IncludeWorktree` is supplied.
