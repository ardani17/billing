package adapter

import "testing"

func TestParseRouterOSMajor(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    RouterOSMajor
	}{
		{name: "v6_long_term_testing", version: "6.49.18 (long-term)", want: RouterOSv6},
		{name: "v7_plain", version: "7.14.3", want: RouterOSv7},
		{name: "v7_with_prefix", version: "RouterOS 7.14.3", want: RouterOSv7},
		{name: "v7_with_whitespace", version: " 7.14.3 ", want: RouterOSv7},
		{name: "empty", version: "", want: RouterOSUnknown},
		{name: "malformed", version: "stable-long-term", want: RouterOSUnknown},
		{name: "unsupported_major", version: "5.26", want: RouterOSUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseRouterOSMajor(tt.version); got != tt.want {
				t.Fatalf("ParseRouterOSMajor(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestCapabilitiesFor(t *testing.T) {
	tests := []struct {
		version           string
		wantMajor         RouterOSMajor
		wantWireGuard     bool
		wantV7CompatCheck bool
	}{
		{version: "6.49.18 (long-term)", wantMajor: RouterOSv6, wantWireGuard: false, wantV7CompatCheck: false},
		{version: "7.14.3", wantMajor: RouterOSv7, wantWireGuard: true, wantV7CompatCheck: true},
		{version: "", wantMajor: RouterOSUnknown, wantWireGuard: false, wantV7CompatCheck: false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := CapabilitiesFor(tt.version)
			if got.Major != tt.wantMajor {
				t.Fatalf("CapabilitiesFor(%q).Major = %v, want %v", tt.version, got.Major, tt.wantMajor)
			}
			if got.SupportsWireGuard != tt.wantWireGuard {
				t.Fatalf("CapabilitiesFor(%q).SupportsWireGuard = %v, want %v", tt.version, got.SupportsWireGuard, tt.wantWireGuard)
			}
		})
	}
}
