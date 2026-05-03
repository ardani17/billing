// Package adapter — Factory untuk membuat instance RouterOSAdapter berdasarkan mode.
// Mode "mock" menggunakan MockAdapter, mode "live" menggunakan LiveAdapter.
// Default ke MockAdapter jika mode tidak dikenali (aman untuk development).
package adapter

// NewAdapter membuat instance RouterOSAdapter berdasarkan mode yang diberikan.
// Mode "live" mengembalikan LiveAdapter untuk koneksi ke router fisik.
// Mode "mock" atau mode lainnya mengembalikan MockAdapter (default aman untuk development).
func NewAdapter(mode string) RouterOSAdapter {
	switch mode {
	case "live":
		return NewLiveAdapter()
	default:
		// Default ke MockAdapter untuk keamanan saat development.
		return NewMockAdapter()
	}
}
