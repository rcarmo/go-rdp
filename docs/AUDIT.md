# Licensing Audit

## Purpose
This document summarizes the state of GPL-origin code in this repository by analyzing the earliest commits in *this* git history and the “GPL removal” refactor.

## Key Findings (Summary)
- The repository **started with GPL licensing**, evidenced by the original `LICENSE` file in early history.
- A **major refactor** explicitly titled *“major refactor to remove GPL code”* was committed on **2026‑01‑19** (commit `d298be7`).
- The `LICENSE` file was then **changed from GPLv3 to MIT** on **2026‑01‑20** (commit `f01d412`).
- Comparing the tree *before* the refactor to the current tree shows **only a small set of shared paths**, primarily build/config/docs scaffolding, not core implementation files.
- The current codebase references FreeRDP/xrdp in comments and documentation **as compatibility references**, not as copied source files.

## Evidence from Git History

### 1) Initial GPL License
Early history includes a GPLv3 license file (commit `3c95fea`):
```
GNU GENERAL PUBLIC LICENSE
Version 3, 29 June 2007
...
```

### 2) GPL Removal Refactor
Commit `d298be7` (2026‑01‑19) is explicitly titled:
```
major refactor to remove GPL code
```
This commit introduces a large volume of new files and deletes large volumes of prior code (see commit stats via `git show --stat d298be7`).

### 3) License Change to MIT
Commit `f01d412` (2026‑01‑20) updates `LICENSE` to MIT:
```
MIT License
Copyright (c) 2026 Rui Carmo
...
```

## File-Level Replacement Evidence
To determine how much of the pre‑refactor tree remains, we compared the tree *before* `d298be7` with the current tree. Only **16 files** are shared, and they are mostly configuration and documentation:

```
cmd/server/main.go
cmd/server/main_test.go
Dockerfile
docker-compose.yml
Makefile
README.md
LICENSE
go.mod
go.sum
docs/configuration.md
docs/debugging.md
docs/index.md
.github/workflows/ci-cd.yml
.gitignore
.dockerignore
web/web_integration_test.go
```

Notably, the core implementation packages (`internal/*`), codec logic, and protocol handling were **added after** the refactor and do **not** overlap with the pre‑refactor tree.

## Remaining Upstream References
The current codebase contains references to FreeRDP and xrdp in comments and documentation (e.g., RemoteFX and NSCodec audits). These references appear to be **algorithmic compatibility notes** rather than direct GPL code reuse.

Examples:
- `internal/codec/rfx/AUDIT.md` (FreeRDP compatibility audit)
- `docs/REMOTEFX.md` (reference implementation notes)

## Conclusion
Based on the repository’s own git history, the GPL‑licensed code present early in the project was **removed and replaced** by the refactor in commit `d298be7`, and the project was re‑licensed under MIT in `f01d412`. The remaining files that overlap with the pre‑refactor tree are limited to configuration, documentation, or top‑level scaffolding, not implementation code.

If deeper verification is required (e.g., similarity analysis against specific GPL source files), we can perform a diff‑based comparison against those sources if provided.
