# Overlay boundary gate

`check-custom-overlay-boundaries.ps1` checks downstream changes against an
explicit Overlay freeze point. It intentionally does not use `origin/main` as
the baseline: upstream synchronization can legitimately change shared files.

The default baseline is the named tag
`baseline/v0.1.164-before-overlay`. For an incremental migration, pass the
reviewed Overlay checkpoint explicitly.

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

The gate allows code under `backend/internal/custom/**`,
`frontend/src/custom/**`, generated Ent files, custom-named migrations,
documentation, reviewed fixed integration points, and an exact published-history
set. The reviewed cleanup set exists only for files whose net change removes old
Overlay behavior but whose formatting makes Git report replacement lines. Any new
business path outside these categories fails the gate.

When no commits exist after the selected baseline, the script prints that the
committed audit is not applicable rather than claiming a full pass. The
worktree is excluded unless `-IncludeWorktree` is supplied.
