---
name: litelens-rebrand-casing
description: "LiteLens -> Litelens display-name rebrand (2026-08-17) — what changed, what was deliberately left alone, and why"
metadata: 
  node_type: memory
  type: project
  originSessionId: e8ef533c-3d22-4012-a9de-adbf3bf879fe
  modified: 2026-08-17T15:17:06.228Z
---

Display brand name changed from "LiteLens" to "Litelens" (capital L, rest lowercase) across UI text, docs, comments, the macOS `.app` bundle path, and the Homebrew cask `name`/caveats field.

**Why:** User-requested rebrand via `/agent-team --feature-dev`. The Go module path (`github.com/litelensapp/litelens`), package identifiers, binary names, storage dir (`~/.litelens`), and env vars (`LITELENS_*`) were kept lowercase — they're implementation details, not display text, and the module path was never mixed-case to begin with.

**Winget `PackageIdentifier` was also renamed** to `litelensapp.Litelens` (was initially left as mixed-case `litelensapp.LiteLens` since it looked like a fixed external identifier — but Winget publishing per `.claude/plans/winget-publishing.md` has never actually been submitted, so there are no real users depending on the old casing yet). Updated consistently in `frontend/src/app/updater/UpdateModal.tsx:30`, `internal/updater/install_source_test.go`, and `.claude/plans/winget-publishing.md` (including the `manifests/l/litelensapp/Litelens/` path convention). If Winget publishing ever ships with this identifier, it becomes a real external contract — don't casually re-case it after that point.

**Deliberately left as "LiteLens" (mixed case), do not "fix" this:**
- `.claude/memory/review-findings.md:13` — historical description of an old test fixture path, kept verbatim for accuracy of the incident record.

**macOS `.app` bundle path was intentionally renamed** from `/Applications/LiteLens.app` to `/Applications/Litelens.app` in `internal/app/updater.go`, `scripts/install.sh`, `scripts/uninstall.sh` — this was a real user-facing install path for a shipped app, so it's a breaking-ish change for existing installs. User explicitly approved this over the safer "leave path as-is" option. To avoid orphaning existing installs, `install.sh`/`uninstall.sh` now also detect/remove the old `/Applications/LiteLens.app` path as a fallback (see "pre-rebrand install path" comments in those scripts). If a user reports a stray old `.app` after upgrading, check those fallback branches first — see [[development_workflow]].

**Homebrew cask** (`.github/workflows/job-publish-homebrew.yml`): only the `name "Litelens"` display field and caveats text changed. The cask slug (`cask "litelens"`) and the `app "litelens.app"` stanza stayed lowercase — those must match the actual build output (`build/bin/litelens.app`, per `wails.json`'s `outputfilename: "litelens"`) and the `brew install litelens` command; renaming them would break the published formula.
