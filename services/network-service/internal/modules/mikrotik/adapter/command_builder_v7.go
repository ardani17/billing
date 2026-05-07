// Paket adapter menyediakan implementasi CommandBuilder untuk RouterOS v7.
// File ini meng-embed commandBuilderV6 dan meng-override method yang berbeda
// antara RouterOS v6 dan v7. Untuk operasi PPPoE, perbedaannya minimal -
// struct ini disiapkan untuk extensibility di masa depan.
package adapter

import (
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// commandBuilderV7 mengimplementasikan domain.CommandBuilder untuk RouterOS v7.
// Meng-embed commandBuilderV6 karena sebagian besar perintah PPPoE identik.
// Method yang berbeda di v7 di-override secara eksplisit.
type commandBuilderV7 struct {
	commandBuilderV6
}

// SetSecret membangun perintah /ppp/secret/atur untuk RouterOS v7.
// Di v7, parameter .id digunakan sebagai alternatif =numbers= untuk identifikasi.
// Namun untuk PPPoE secret, =numbers= dengan username tetap berfungsi di v7,
// sehingga perilaku sama dengan v6.
func (b *commandBuilderV7) SetSecret(username string, params map[string]string) (string, map[string]string) {
	return b.commandBuilderV6.SetSecret(username, params)
}

// SetProfile membangun perintah /ppp/profile/atur untuk RouterOS v7.
// Sama dengan v6 - =numbers= dengan nama profile tetap berfungsi di v7.
func (b *commandBuilderV7) SetProfile(name string, params map[string]string) (string, map[string]string) {
	return b.commandBuilderV6.SetProfile(name, params)
}

// CreateSimpleQueue membangun perintah /queue/simple/add untuk RouterOS v7.
// Di v7, parameter "target" tetap sama dengan v6 untuk simple queue.
func (b *commandBuilderV7) CreateSimpleQueue(params domain.SimpleQueueParams) (string, map[string]string) {
	return b.commandBuilderV6.CreateSimpleQueue(params)
}

// SetSimpleQueue membangun perintah /queue/simple/atur untuk RouterOS v7.
// Sama dengan v6 - =numbers= dengan nama queue tetap berfungsi di v7.
func (b *commandBuilderV7) SetSimpleQueue(name string, params map[string]string) (string, map[string]string) {
	return b.commandBuilderV6.SetSimpleQueue(name, params)
}

// ResetSimpleQueueCounters membangun perintah /queue/simple/reset-counters untuk RouterOS v7.
// Path dan parameter sama dengan v6.
func (b *commandBuilderV7) ResetSimpleQueueCounters(name string) (string, map[string]string) {
	return b.commandBuilderV6.ResetSimpleQueueCounters(name)
}
