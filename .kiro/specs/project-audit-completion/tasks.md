# Implementation Plan

- [x] 1. Restore and verify frontend build
  - Identify why `next` is unavailable in the web workspace.
  - Install or document the required dependency setup.
  - Run `npm.cmd --workspace @ispboss/web run build`.
  - Record build result in the audit document.

- [x] 2. Audit all settings routes
  - Review each route under `apps/web/app/settings`.
  - Classify each page as live, partial, placeholder, or deferred.
  - Replace placeholder pages that are in active scope with live pages.
  - Document pages that remain deferred.

- [x] 3. Create permission matrix
  - List role/action combinations for admin, finance, operator, reseller, and read-only users.
  - Map each sensitive backend endpoint to required permission.
  - Ensure frontend actions respect the same permission.
  - Add tests for denied sensitive actions where practical.

- [x] 4. Run core backend tests
  - Run `go test ./...` for `services/billing-api`.
  - Run `go test ./...` for `services/notification`.
  - Fix regressions introduced by audit-completion work.

- [x] 5. Run core UI smoke test
  - Verify login/session.
  - Verify dashboard render.
  - Verify customer list/detail/create/edit.
  - Verify package list/create/edit.
  - Verify notification page.
  - Verify report page.
  - Verify settings index and active settings pages.

- [x] 6. Verify notification integration
  - Trigger or simulate invoice reminder.
  - Confirm notification job/event is created.
  - Confirm template and channel selection.
  - Confirm success/failure status is stored.
  - Document any channel that is configured but not production-ready.

- [x] 7. Synchronize audit and diskusi status
  - Update the audit document with final findings.
  - Mark features as backend done, UI partial, usable, or deferred.
  - Keep MikroTik, OLT, and Map marked deferred per current direction.
  - Link `project-audit-completion` and `financial-completion` as follow-up specs.
