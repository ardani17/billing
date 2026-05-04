# Requirements Document

## Introduction

Dokumen ini mendefinisikan requirements untuk **MikroTik Completion Layer**. Spec ini melanjutkan spec yang sudah selesai: `mikrotik-router`, `mikrotik-pppoe`, dan `mikrotik-vpn`.

Scope utama adalah menutup gap dari `diskusi/08-mikrotik.md` agar modul MikroTik siap digunakan secara operasional oleh Tenant Admin ISP. Fokusnya adalah fitur detail router yang belum lengkap: operational read-only tabs, DHCP/static IP, hotspot, walled garden, backup, firmware tracking, terminal aman, bulk action, audit trail, dan integrasi UI submenu MikroTik.

## Existing Foundation

- Router CRUD, edit, delete, test connection, status summary sudah tersedia.
- RouterOS API live adapter sudah tersedia dan mendukung API-SSL self-signed untuk lab.
- PPPoE user CRUD, enable/disable, delete, disconnect, session live, sync, sync status, worker event, profile sync, dan isolir/unisolir dasar sudah tersedia.
- VPN tunnel backend sudah tersedia cukup luas: CRUD, script generator, auto-config, health monitor, bandwidth store.
- Health checker dan PPPoE sync scheduler sengaja bisa dinonaktifkan agar tidak login API terus-menerus.

## Requirements

### Requirement 1: Operational Read-Only RouterOS Tabs

**User Story:** As a Tenant Admin, I want to inspect live router data from separated MikroTik detail tabs, so that I can troubleshoot router state without opening Winbox.

#### Acceptance Criteria

1. THE Network_Service SHALL expose read-only endpoints for router interfaces, traffic, IP pools, firewall managed rules, and router logs.
2. THE Network_Service SHALL execute these reads only on explicit user action or page load for the selected tab, not through uncontrolled polling.
3. THE Network_Service SHALL support RouterOS v6.49.x and v7 command differences through adapter/command-builder patterns.
4. THE Web App SHALL show the operational tabs under the MikroTik sidebar submenu: Overview, PPPoE users, Sessions, Sync, Traffic, Interfaces, IP Pool, Firewall, Logs.
5. WHEN a router is offline or credentials are invalid, THE Web App SHALL show an inline error state and preserve the rest of the page layout.
6. WHEN a live read succeeds, THE Network_Service SHALL update only safe metadata where useful; it SHALL NOT mutate customer/service configuration.

### Requirement 2: Traffic and Interface Monitoring

**User Story:** As a technician, I want to read interface status and traffic for a router, so that I can diagnose uplink and LAN issues.

#### Acceptance Criteria

1. THE Network_Service SHALL expose `GET /api/v1/mikrotik/routers/:id/interfaces`.
2. THE Network_Service SHALL expose `GET /api/v1/mikrotik/routers/:id/traffic?interfaces=...`.
3. THE interfaces endpoint SHALL return name, type, mtu, mac address, running/disabled status, rx/tx bytes, rx/tx packets where available.
4. THE traffic endpoint SHALL use RouterOS `/interface/monitor-traffic` and return rx/tx bps per interface.
5. THE Web App SHALL let the user choose which interface to refresh; auto-refresh SHALL be optional and disabled by default.

### Requirement 3: IP Pool Usage

**User Story:** As a Tenant Admin, I want to view IP pool usage, so that I can prevent PPPoE/hotspot/DHCP customers from failing due to full pools.

#### Acceptance Criteria

1. THE Network_Service SHALL expose `GET /api/v1/mikrotik/routers/:id/ip-pools`.
2. THE endpoint SHALL return pool name, ranges, used count, estimated total count, usage percentage, and warning level.
3. WHEN pool usage is above 80%, THE response SHALL mark warning level as `warning`.
4. WHEN pool usage is above 90%, THE response SHALL mark warning level as `critical`.
5. THE Web App SHALL display pool usage compactly in the IP Pool submenu.

### Requirement 4: Firewall Managed Rules Read-Only

**User Story:** As a Tenant Admin, I want to inspect firewall rules managed by ISPBoss, so that I can verify isolir and walled garden behavior without editing router firewall directly.

#### Acceptance Criteria

1. THE Network_Service SHALL expose `GET /api/v1/mikrotik/routers/:id/firewall/managed`.
2. THE endpoint SHALL return firewall nat/filter rules and address-lists whose comment/name begins with `ISPBoss:`.
3. THE endpoint SHALL be read-only; no firewall edit/delete endpoint is included in this phase.
4. THE Web App SHALL show firewall rules, address lists, chain, action, disabled state, and comment.

### Requirement 5: Router Logs

**User Story:** As a technician, I want router logs in the dashboard, so that I can inspect PPPoE login/logout and system events quickly.

#### Acceptance Criteria

1. THE Network_Service SHALL expose `GET /api/v1/mikrotik/routers/:id/logs`.
2. THE endpoint SHALL support query filters: topic, search, limit.
3. THE endpoint SHALL cap limit to a safe maximum.
4. THE Web App SHALL show logs in a dedicated submenu with refresh button and filter fields.

### Requirement 6: DHCP Binding Management

**User Story:** As a Tenant Admin, I want DHCP static bindings for customers, so that non-PPPoE customers can receive stable IP addresses.

#### Acceptance Criteria

1. THE Network_Service SHALL expose read endpoints for DHCP servers, leases, static bindings, and networks.
2. THE Network_Service SHALL expose create/update/delete endpoints only for static leases/bindings managed by ISPBoss.
3. THE endpoint SHALL write RouterOS `/ip/dhcp-server/lease` only when user explicitly submits a form or a customer activation event requires it.
4. THE Network_Service SHALL support disabling DHCP bindings for isolir flow.
5. THE Web App SHALL add DHCP submenu with Servers, Leases, Static Bindings, and Networks sections.
6. DHCP networks SHALL be read-only in the dashboard.

### Requirement 7: Static IP Management

**User Story:** As a Tenant Admin, I want static IP customer provisioning, so that customers using full static IP can be controlled through ISPBoss.

#### Acceptance Criteria

1. THE Network_Service SHALL store static IP assignments linked to tenant, router, customer, IP address, queue name, and status.
2. THE Network_Service SHALL provision address-list and optional simple queue for static IP customers.
3. THE Network_Service SHALL isolate static IP customers by moving/removing address-list membership and applying walled garden/block rules.
4. THE Web App SHALL expose Static IP section after DHCP foundation is complete.

### Requirement 8: Hotspot and Voucher Integration

**User Story:** As a Tenant Admin, I want hotspot users generated from vouchers, so that prepaid users can authenticate through MikroTik Hotspot.

#### Acceptance Criteria

1. THE Network_Service SHALL expose hotspot profile and user APIs for RouterOS `/ip/hotspot/user`.
2. THE Network_Service SHALL support voucher activation event from reseller/voucher module.
3. THE Web App SHALL show Hotspot submenu with users, profiles, active sessions, and login page template status.
4. The custom hotspot login page generation SHALL use tenant branding settings.

### Requirement 9: Walled Garden Completion

**User Story:** As a Tenant Admin, I want isolir customers to see a billing page and payment link, so that they can self-restore after payment.

#### Acceptance Criteria

1. THE Network_Service SHALL provide command builders for DNS redirect, HTTP redirect, and block-all plus whitelist isolir methods.
2. THE Billing API SHALL choose isolir method from tenant settings.
3. THE Network_Service SHALL create/update/remove only `ISPBoss:` firewall/address-list entries.
4. THE Web App SHALL show walled garden/firewall status per router.
5. All isolir/unisolir commands SHALL be event-driven and idempotent.

### Requirement 10: Terminal with Safety Controls

**User Story:** As an advanced Tenant Admin or technician, I want a controlled terminal, so that I can run limited RouterOS commands from the dashboard.

#### Acceptance Criteria

1. THE Network_Service SHALL expose `POST /api/v1/mikrotik/routers/:id/terminal/execute`.
2. The terminal SHALL require role authorization for Tenant Admin or Technician.
3. The terminal SHALL reject dangerous commands including system reset, shutdown, user removal, file removal, and unrestricted script execution.
4. Every terminal command SHALL be written to append-only audit log with user id, tenant id, router id, command, result status, timestamp, and IP address.
5. The Web App SHALL show a warning state and command history.

### Requirement 11: Backup and Firmware Tracking

**User Story:** As a Tenant Admin, I want router backup history and firmware tracking, so that I can recover configs and detect outdated routers.

#### Acceptance Criteria

1. THE Network_Service SHALL support manual backup export as `.rsc`.
2. THE Network_Service SHALL store backup metadata and file path/object key.
3. THE Network_Service SHALL support scheduled weekly backup when enabled.
4. THE Network_Service SHALL retain the latest 10 backups by default.
5. Restore SHALL require explicit confirmation and SHALL be limited to Tenant Admin.
6. Firmware tracking SHALL read current version and mark outdated routers; it SHALL NOT auto-upgrade firmware.

### Requirement 12: Bulk Actions

**User Story:** As a Tenant Admin, I want safe bulk operations, so that I can operate multiple routers without repetitive manual work.

#### Acceptance Criteria

1. THE Network_Service SHALL support bulk sync, bulk backup, firmware check, and export status.
2. Bulk operations SHALL run async through queue jobs and expose job status.
3. Bulk operations SHALL be per tenant and respect RBAC.
4. THE Web App SHALL show confirmation and progress summary.

### Requirement 13: Audit Trail

**User Story:** As an owner/admin, I want every router-changing command audited, so that operational mistakes can be traced.

#### Acceptance Criteria

1. THE Network_Service SHALL write audit records for router-changing commands: PPPoE write, DHCP binding write, static IP write, firewall write, terminal command, backup restore, VPN auto-config.
2. Read-only live inspections SHALL be logged at debug/info level but do not need command audit rows unless configured.
3. Audit records SHALL be append-only and tenant scoped.
4. The Web App SHALL expose audit data through settings/audit-log or router detail audit tab in a later phase.

### Requirement 14: Implementation Safety

**User Story:** As the application owner, I want real MikroTik integration without accidental repeated logins or destructive writes, so that the test CHR remains safe while development continues.

#### Acceptance Criteria

1. New live read/write features SHALL use on-demand execution by default.
2. No new scheduler SHALL be enabled by default unless explicitly configured through environment variable.
3. All RouterOS write operations SHALL be idempotent where possible.
4. All destructive operations SHALL require explicit confirmation in UI and backend.
5. Integration tests against CHR SHALL be grouped and opt-in via environment variables.
