# TODO MikroTik Completion

## Phase 1 - Operational Read-Only

- [x] Interfaces, traffic, IP pools, firewall managed, logs.
- [x] Proxy API dan submenu UI.
- [x] Build, service test, CHR smoke test.
- [ ] Parser unit tests.
- [ ] Handler unit tests.
- [ ] Mobile visual verification.

## Phase 2 - DHCP

- [x] DHCP servers, leases, static bindings, networks read model.
- [x] Static binding create/update/delete with explicit user action.
- [x] Managed binding storage in `dhcp_bindings`.
- [x] RouterOS write must be idempotent and comment-prefixed with `ISPBoss:dhcp:`.
- [x] Web submenu DHCP with read sections and binding form.
- [x] Real CHR smoke test: read, create test binding, update/disable, delete cleanup.

## Phase 3 - Static IP

- [ ] `static_ip_assignments` storage.
- [ ] Address-list and optional simple queue provisioning.
- [ ] Isolate/unisolate static IP customers.
- [ ] Static IP submenu UI and tests.

## Phase 4 - Walled Garden

- [ ] DNS redirect, HTTP redirect, and block-all whitelist builders.
- [ ] Tenant setting lookup for isolir method.
- [ ] Managed firewall status in UI.
- [ ] Idempotency tests.

## Phase 5 - Hotspot

- [ ] Hotspot user/profile/active endpoints.
- [ ] Voucher activation integration.
- [ ] Hotspot submenu UI.
- [ ] Branded login template generation.

## Phase 6 - Terminal And Audit

- [ ] `mikrotik_command_audit_logs`.
- [ ] Safe command validator and denylist.
- [ ] Terminal execute endpoint and UI.
- [ ] Append-only audit for every attempt.

## Phase 7 - Backup And Firmware

- [ ] Manual export backup.
- [ ] Backup metadata and retention.
- [ ] Restore with confirmation.
- [ ] Firmware read and outdated warning.
- [ ] Backup/Firmware UI.

## Phase 8 - Bulk Actions

- [ ] Bulk jobs table and status model.
- [ ] Bulk sync, backup, firmware check, export status.
- [ ] UI confirmation and progress.

## Final Hardening

- [ ] RBAC review for all write paths.
- [ ] Scheduler defaults review.
- [ ] Full tests and web build.
- [ ] Real CHR opt-in smoke tests.
- [ ] Update project report.
