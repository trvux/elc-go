Before any Go coding, review, debugging, troubleshooting, or setup task, load the `samber/cc-skills-golang@golang-how-to` skill first — it routes to whichever other Go skills the task actually needs, so only the relevant ones get loaded instead of all of them every time.

## Domain entity: consolidate per-field UpdateX() into one Update(input)

Many small CRUD modules under `internal/` (branch, contact, tag, author, event, brand, page, group, service, ...) still have one `UpdateX()`/`SetX()` domain method per field, copied from before this convention existed. `internal/branch` was refactored to a single `Update(input UpdateXInput) error` method instead — see `docs/rfc/2026-08-18-branch-domain-consolidate-update.md` for the pattern, the exact behavior-preservation pitfall it caught (don't bump `updatedAt` on a no-op call — track whether any field actually changed), and the review process used.

**Do not proactively sweep the remaining modules to this pattern** — the LOC payoff per module is small and the user decided against a dedicated cleanup pass (2026-08-18). Instead, apply it **lazily**: whenever a module's domain entity is touched anyway for an unrelated feature or bugfix, and that entity still has the old per-field UpdateX() shape, fold this consolidation into that same change rather than opening a separate refactor PR for it.

## Keep RFC/doc status current when finishing the work it describes

When a `docs/rfc/*.md` (or any planning doc under `docs/`) is fully or partially
implemented, update that file's `Status` field in the **same commit/PR** that
finishes the work — don't leave it saying Draft/In-progress after the work is
done. Stale status on a planning doc is itself clutter: it misleads whoever
(human or AI) reads it next into re-investigating or re-doing work that's
already finished. This was a recurring source of repo cruft (found 2026-09-01:
`docs/rfc/2026-08-18-backend-code-cleanup.md` said Draft while every item in
it had already shipped).