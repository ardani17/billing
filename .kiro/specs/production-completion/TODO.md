# TODO Production Completion

## Current Status

- Local stack healthy.
- Go tests for billing-api, network-service, notification passed.
- Web build passed.
- MikroTik CHR live integration works on-demand.

## Now

- [x] MikroTik Backup/Firmware.
- [x] MikroTik Bulk Actions.

## Next

- [ ] Tenant Settings Persistence.
- [ ] Notification Production.
- [ ] Tenant Admin End-to-End smoke.
- [ ] OLT Real Validation.
- [ ] Super Admin Hardening.
- [ ] Production Hardening.

## Review Notes

- Do not re-enable periodic router login by default.
- Do not add fake operational data.
- Use audit/event logs for write actions.
- Clear `.next` and restart dev server after major Next build or route changes.
- CHR backup smoke currently uses read-only inventory fallback because RouterOS denies `export file` for the API user; grant write/ftp policy for full importable export.
