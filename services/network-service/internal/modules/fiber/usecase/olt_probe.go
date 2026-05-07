package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

const (
	oltProbeSysDescr  = "1.3.6.1.2.1.1.1.0"
	oltProbeSysUpTime = "1.3.6.1.2.1.1.3.0"
	oltProbeSysName   = "1.3.6.1.2.1.1.5.0"
)

// probeOLTSystemInfo membaca OID standar agar brand/model bisa dideteksi sebelum adapter brand dibuat.
func (m *oltManager) probeOLTSystemInfo(ctx context.Context, cfg domain.SNMPConfig) (*domain.OLTSystemInfo, error) {
	results, err := m.snmpConn.Get(ctx, cfg, []string{oltProbeSysDescr, oltProbeSysUpTime, oltProbeSysName})
	if err != nil {
		return nil, err
	}

	info := &domain.OLTSystemInfo{}
	for _, r := range results {
		switch normalizeProbeOID(r.OID) {
		case oltProbeSysDescr:
			info.SysDescr = probeResultToString(r)
		case oltProbeSysUpTime:
			info.Uptime = probeResultToInt64(r) / 100
		case oltProbeSysName:
			info.SysName = probeResultToString(r)
		}
	}

	info.Brand = domain.DetectBrand(info.SysDescr)
	if info.Brand == "" {
		return info, domain.ErrBrandDetectionFailed
	}
	info.Model, info.FirmwareVersion = detectOLTModelFirmware(info.Brand, info.SysDescr)
	return info, nil
}

func normalizeProbeOID(oid string) string {
	return strings.TrimPrefix(oid, ".")
}

func probeResultToString(r domain.SNMPResult) string {
	if r.Value == nil {
		return ""
	}
	switch v := r.Value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func probeResultToInt64(r domain.SNMPResult) int64 {
	if r.Value == nil {
		return 0
	}
	switch v := r.Value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	default:
		return 0
	}
}

func detectOLTModelFirmware(brand domain.OLTBrand, sysDescr string) (model, firmware string) {
	upper := strings.ToUpper(sysDescr)
	switch brand {
	case domain.BrandZTE:
		for _, candidate := range []string{"C600", "C320", "C300", "C220"} {
			if strings.Contains(upper, candidate) {
				model = candidate
				break
			}
		}
	case domain.BrandHuawei:
		for _, candidate := range []string{"MA5800", "MA5683", "MA5608", "MA5600"} {
			if strings.Contains(upper, candidate) {
				model = candidate
				break
			}
		}
	case domain.BrandFiberHome:
		for _, candidate := range []string{"AN5516", "AN6000"} {
			if strings.Contains(upper, candidate) {
				model = candidate
				break
			}
		}
	case domain.BrandVSOL:
		for _, candidate := range []string{"V1600G", "V1600D", "V1600"} {
			if strings.Contains(upper, candidate) {
				model = candidate
				break
			}
		}
	}

	for _, part := range strings.Fields(sysDescr) {
		if len(part) > 1 && (part[0] == 'V' || part[0] == 'v') && part[1] >= '0' && part[1] <= '9' {
			firmware = part
			break
		}
	}
	return model, firmware
}
