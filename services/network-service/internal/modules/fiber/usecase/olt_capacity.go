// Package usecase berisi implementasi business logic untuk network-service.
// File ini mengimplementasikan GetCapacity pada oltManager untuk capacity planning OLT.
package usecase

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// maxONTPerPort adalah jumlah maksimum ONT per PON port (standar GPON).
const maxONTPerPort = 64

// warningThreshold adalah batas utilisasi port (90%) untuk menampilkan warning.
const warningThreshold = 90.0

// GetCapacity mengambil data capacity planning untuk satu OLT.
// Mengambil OLT dari repo, ambil semua PON port via adapter,
// lalu hitung total/active ports, total/used/available ONT slots,
// utilization percent, dan per-port breakdown dengan warning jika > 90%.
func (m *oltManager) GetCapacity(ctx context.Context, oltID string) (*domain.OLTCapacity, error) {
	olt, err := m.oltRepo.GetByID(ctx, oltID)
	if err != nil {
		return nil, err
	}

	// Ambil status semua PON port via adapter
	ports, portErr := m.fetchPONPorts(ctx, olt)

	// Jika gagal ambil dari adapter, gunakan data statis dari DB
	if portErr != nil {
		log.Warn().Err(portErr).Str("olt_id", oltID).
			Msg("gagal ambil PON ports untuk capacity, gunakan data DB")
		return m.buildStaticCapacity(olt), nil
	}

	return m.calculateCapacity(ports), nil
}

// fetchPONPorts mengambil status PON port dari adapter OLT.
func (m *oltManager) fetchPONPorts(ctx context.Context, olt *domain.OLT) ([]domain.PONPortStatus, error) {
	adapter, err := m.createAdapter(ctx, olt)
	if err != nil {
		return nil, err
	}
	return adapter.GetAllPONPorts(ctx)
}

// calculateCapacity menghitung kapasitas OLT dari data PON port aktual.
func (m *oltManager) calculateCapacity(ports []domain.PONPortStatus) *domain.OLTCapacity {
	totalPorts := len(ports)
	activePorts := 0
	usedSlots := 0
	breakdown := make([]domain.PortCapacity, 0, totalPorts)

	for _, port := range ports {
		// Hitung port aktif (oper_status = up)
		if port.OperStatus == "up" {
			activePorts++
		}
		usedSlots += port.ONTCount

		// Hitung utilisasi per port
		portUtil := 0.0
		if maxONTPerPort > 0 {
			portUtil = float64(port.ONTCount) / float64(maxONTPerPort) * 100
		}

		pc := domain.PortCapacity{
			PortIndex:          port.PortIndex,
			ONTCount:           port.ONTCount,
			MaxONTPerPort:      maxONTPerPort,
			UtilizationPercent: portUtil,
		}

		// Warning jika utilisasi melebihi 90%
		if portUtil > warningThreshold {
			pc.Warning = fmt.Sprintf(
				"Port %d utilisasi %.1f%% (>90%%)", port.PortIndex, portUtil,
			)
		}

		breakdown = append(breakdown, pc)
	}

	totalSlots := totalPorts * maxONTPerPort
	availableSlots := totalSlots - usedSlots
	if availableSlots < 0 {
		availableSlots = 0
	}

	utilPercent := 0.0
	if totalSlots > 0 {
		utilPercent = float64(usedSlots) / float64(totalSlots) * 100
	}

	// Growth rate dan estimated months - placeholder, butuh data historis
	growthRate := 0.0
	estimatedMonths := 0.0

	return &domain.OLTCapacity{
		TotalPONPorts:            totalPorts,
		ActivePONPorts:           activePorts,
		TotalONTSlots:            totalSlots,
		UsedONTSlots:             usedSlots,
		AvailableONTSlots:        availableSlots,
		UtilizationPercent:       utilPercent,
		GrowthRatePerMonth:       growthRate,
		EstimatedMonthsRemaining: estimatedMonths,
		PortBreakdown:            breakdown,
	}
}

// buildStaticCapacity membangun kapasitas dari data statis OLT di database.
// Digunakan sebagai cadangan jika adapter tidak tersedia.
func (m *oltManager) buildStaticCapacity(olt *domain.OLT) *domain.OLTCapacity {
	totalSlots := olt.PONPortCount * maxONTPerPort
	availableSlots := totalSlots - olt.TotalONTCount
	if availableSlots < 0 {
		availableSlots = 0
	}

	utilPercent := 0.0
	if totalSlots > 0 {
		utilPercent = float64(olt.TotalONTCount) / float64(totalSlots) * 100
	}

	return &domain.OLTCapacity{
		TotalPONPorts:      olt.PONPortCount,
		TotalONTSlots:      totalSlots,
		UsedONTSlots:       olt.TotalONTCount,
		AvailableONTSlots:  availableSlots,
		UtilizationPercent: utilPercent,
	}
}
