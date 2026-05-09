package usecase

import (
	"testing"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"pgregory.net/rapid"
)

// allChannels berisi semua channel yang tersedia.
var allChannels = []domain.Channel{
	domain.ChannelWhatsApp,
	domain.ChannelSMS,
	domain.ChannelEmail,
}

// genChannelSubset menghasilkan subset acak (non-kosong) dari daftar channel.
// Menggunakan bitmask 1..7 untuk menghasilkan semua kombinasi non-kosong dari 3 channel.
func genChannelSubset(t *rapid.T, label string) []domain.Channel {
	mask := rapid.IntRange(1, 7).Draw(t, label)
	var out []domain.Channel
	for i, ch := range allChannels {
		if mask&(1<<i) != 0 {
			out = append(out, ch)
		}
	}
	return out
}

// genEnabledConfigs menghasilkan daftar NotificationConfig dengan channel yang di-enable
// berdasarkan subset acak dari daftar channel.
func genEnabledConfigs(t *rapid.T, label string) []*domain.NotificationConfig {
	chs := genChannelSubset(t, label)
	var cfgs []*domain.NotificationConfig
	for _, ch := range chs {
		cfgs = append(cfgs, &domain.NotificationConfig{
			Channel:   ch,
			IsEnabled: true,
		})
	}
	return cfgs
}

// channelSet mengembalikan atur dari slice channel.
func channelSet(chs []domain.Channel) map[domain.Channel]bool {
	s := make(map[domain.Channel]bool)
	for _, ch := range chs {
		s[ch] = true
	}
	return s
}

// enabledSet mengembalikan atur channel yang di-enable dari daftar config.
func enabledSet(cfgs []*domain.NotificationConfig) map[domain.Channel]bool {
	s := make(map[domain.Channel]bool)
	for _, c := range cfgs {
		if c.IsEnabled {
			s[c.Channel] = true
		}
	}
	return s
}

// **Memvalidasi: Kebutuhan 7.4**
//
// Untuk setiap daftar prioritas channel tenant dan konfigurasi channel template,
// channel yang dipilih (elemen pertama dari hasil pickChannels) HARUS merupakan
// channel pertama dalam daftar prioritas yang juga ada di daftar channel template
// DAN memiliki konfigurasi provider yang aktif (is_enabled=true).
func TestProperty_ChannelSelectionRespectsPriorityOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		prio := rapid.Permutation(allChannels).Draw(t, "priority")
		tmplCh := genChannelSubset(t, "templateChannels")
		cfgs := genEnabledConfigs(t, "enabledConfigs")

		// Panggil pickChannels pada zero-value pipeline
		var pipeline DeliveryPipeline
		result := pipeline.pickChannels(prio, tmplCh, cfgs)

		tmplSet := channelSet(tmplCh)
		enSet := enabledSet(cfgs)

		var expected []domain.Channel
		for _, ch := range prio {
			if tmplSet[ch] && enSet[ch] {
				expected = append(expected, ch)
			}
		}

		// Verifikasi panjang hasil sama
		if len(result) != len(expected) {
			t.Fatalf(
				"pickChannels panjang berbeda:\n  prio=%v tmplCh=%v enabled=%v\n  expected=%v (len %d)\n  got=%v (len %d)",
				prio, tmplCh, enSet, expected, len(expected), result, len(result),
			)
		}

		// Verifikasi urutan hasil sama persis
		for i := range expected {
			if result[i] != expected[i] {
				t.Fatalf(
					"pickChannels urutan salah di index %d:\n  prio=%v tmplCh=%v enabled=%v\n  expected=%v\n  got=%v",
					i, prio, tmplCh, enSet, expected, result,
				)
			}
		}

		// Verifikasi elemen pertama (jika ada) adalah channel pertama di prioritas
		// yang memenuhi kedua syarat (template + enabled)
		if len(result) > 0 {
			firstResult := result[0]
			var firstExpected domain.Channel
			for _, ch := range prio {
				if tmplSet[ch] && enSet[ch] {
					firstExpected = ch
					break
				}
			}
			if firstResult != firstExpected {
				t.Fatalf(
					"Channel pertama salah:\n  expected=%v got=%v\n  prio=%v tmplCh=%v enabled=%v",
					firstExpected, firstResult, prio, tmplCh, enSet,
				)
			}
		}
	})
}
