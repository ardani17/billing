package adapter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// nonEmptyAlphanumString generates a non-empty string suitable untuk RouterOS parameters.
func nonEmptyAlphanumString() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		chars := rapid.SliceOfN(
			rapid.ByteRange('a', 'z'),
			1, 30,
		).Draw(t, "chars")
		return string(chars)
	})
}

func ipv4String() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		a := rapid.IntRange(1, 254).Draw(t, "octet1")
		b := rapid.IntRange(0, 255).Draw(t, "octet2")
		c := rapid.IntRange(0, 255).Draw(t, "octet3")
		d := rapid.IntRange(1, 254).Draw(t, "octet4")
		return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d)
	})
}

// rateLimitString generates a rate-limit string like "50M/25M".
func rateLimitString() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		dl := rapid.IntRange(1, 1000).Draw(t, "download")
		ul := rapid.IntRange(1, 1000).Draw(t, "upload")
		return fmt.Sprintf("%dM/%dM", dl, ul)
	})
}

// =============================================================================
// =============================================================================

// PPPoESecretParams (non-empty name, password, service, profile, comment), the
// command builder produces:
// - Command string is "/ppp/secret/add"
//
// **Memvalidasi: Kebutuhan 3.2**
func TestProperty_PPPoESecretCommandBuilderCompleteness(t *testing.T) {
	builders := []struct {
		name    string
		builder domain.CommandBuilder
	}{
		{"v6", NewCommandBuilder("6.49.10")},
		{"v7", NewCommandBuilder("7.14.3")},
	}

	for _, b := range builders {
		t.Run(b.name, func(t *testing.T) {
			builder := b.builder
			rapid.Check(t, func(t *rapid.T) {
				params := domain.PPPoESecretParams{
					Name:     nonEmptyAlphanumString().Draw(t, "name"),
					Password: nonEmptyAlphanumString().Draw(t, "password"),
					Service:  nonEmptyAlphanumString().Draw(t, "service"),
					Profile:  nonEmptyAlphanumString().Draw(t, "profile"),
					Comment:  nonEmptyAlphanumString().Draw(t, "comment"),
				}

				command, args := builder.CreateSecret(params)

				if command != "/ppp/secret/add" {
					t.Errorf("command = %q, want /ppp/secret/add", command)
				}

				requiredKeys := []string{"=name", "=password", "=service", "=profile", "=comment"}
				for _, key := range requiredKeys {
					if _, ok := args[key]; !ok {
						t.Errorf("args missing required key %q", key)
					}
				}

				if args["=name"] != params.Name {
					t.Errorf("args[=name] = %q, want %q", args["=name"], params.Name)
				}
				if args["=password"] != params.Password {
					t.Errorf("args[=password] = %q, want %q", args["=password"], params.Password)
				}
				if args["=service"] != params.Service {
					t.Errorf("args[=service] = %q, want %q", args["=service"], params.Service)
				}
				if args["=profile"] != params.Profile {
					t.Errorf("args[=profile] = %q, want %q", args["=profile"], params.Profile)
				}
				if args["=comment"] != params.Comment {
					t.Errorf("args[=comment] = %q, want %q", args["=comment"], params.Comment)
				}
			})
		})
	}
}

// =============================================================================
// =============================================================================

// "=local-address", "=rate-limit", "=only-one".
//
// **Memvalidasi: Kebutuhan 6.2, 6.3**
func TestProperty_ProfileCommandBuilderConditionalBurst(t *testing.T) {
	builders := []struct {
		name    string
		builder domain.CommandBuilder
	}{
		{"v6", NewCommandBuilder("6.49.10")},
		{"v7", NewCommandBuilder("7.14.3")},
	}

	for _, b := range builders {
		t.Run(b.name, func(t *testing.T) {
			builder := b.builder
			rapid.Check(t, func(t *rapid.T) {
				hasBurst := rapid.Bool().Draw(t, "hasBurst")

				params := domain.PPPoEProfileParams{
					Name:          nonEmptyAlphanumString().Draw(t, "name"),
					LocalAddress:  ipv4String().Draw(t, "localAddress"),
					RemoteAddress: nonEmptyAlphanumString().Draw(t, "remoteAddress"),
					RateLimit:     rateLimitString().Draw(t, "rateLimit"),
					OnlyOne:       rapid.SampledFrom([]string{"yes", "no"}).Draw(t, "onlyOne"),
				}

				if hasBurst {
					params.BurstLimit = rateLimitString().Draw(t, "burstLimit")
					params.BurstThreshold = rateLimitString().Draw(t, "burstThreshold")
					params.BurstTime = nonEmptyAlphanumString().Draw(t, "burstTime")
				}

				command, args := builder.CreateProfile(params)

				if command != "/ppp/profile/add" {
					t.Errorf("command = %q, want /ppp/profile/add", command)
				}

				alwaysRequired := []string{"=name", "=local-address", "=rate-limit", "=only-one"}
				for _, key := range alwaysRequired {
					if _, ok := args[key]; !ok {
						t.Errorf("args missing required key %q", key)
					}
				}

				if args["=name"] != params.Name {
					t.Errorf("args[=name] = %q, want %q", args["=name"], params.Name)
				}
				if args["=local-address"] != params.LocalAddress {
					t.Errorf("args[=local-address] = %q, want %q", args["=local-address"], params.LocalAddress)
				}
				if args["=rate-limit"] != params.RateLimit {
					t.Errorf("args[=rate-limit] = %q, want %q", args["=rate-limit"], params.RateLimit)
				}
				if args["=only-one"] != params.OnlyOne {
					t.Errorf("args[=only-one] = %q, want %q", args["=only-one"], params.OnlyOne)
				}

				burstKeys := []string{"=burst-limit", "=burst-threshold", "=burst-time"}

				if hasBurst {
					for _, key := range burstKeys {
						if _, ok := args[key]; !ok {
							t.Errorf("burst enabled but args missing key %q", key)
						}
					}
					if args["=burst-limit"] != params.BurstLimit {
						t.Errorf("args[=burst-limit] = %q, want %q", args["=burst-limit"], params.BurstLimit)
					}
					if args["=burst-threshold"] != params.BurstThreshold {
						t.Errorf("args[=burst-threshold] = %q, want %q", args["=burst-threshold"], params.BurstThreshold)
					}
					if args["=burst-time"] != params.BurstTime {
						t.Errorf("args[=burst-time] = %q, want %q", args["=burst-time"], params.BurstTime)
					}
				} else {
					for _, key := range burstKeys {
						if _, ok := args[key]; ok {
							t.Errorf("burst disabled but args contains key %q", key)
						}
					}
				}
			})
		})
	}
}

// =============================================================================
// =============================================================================

// TestProperty_VersionAwareCommandPathSelection verifies that:
//
// **Memvalidasi: Kebutuhan 13.2, 13.3**
func TestProperty_VersionAwareCommandPathSelection(t *testing.T) {
	t.Run("real_v6_testing_version", func(t *testing.T) {
		builder := NewCommandBuilder("6.49.18 (long-term)")
		if _, ok := builder.(*commandBuilderV6); !ok {
			t.Fatalf("NewCommandBuilder(real v6 testing version) returned %T, want *commandBuilderV6", builder)
		}
		command, args := builder.SetSecret("testing-user", map[string]string{
			"=profile":  "default",
			"=disabled": "false",
		})
		if command != "/ppp/secret/set" {
			t.Fatalf("command = %q, want /ppp/secret/set", command)
		}
		if args["=profile"] != "default" || args["=disabled"] != "false" {
			t.Fatalf("v6 SetSecret must preserve RouterOS parameter keys, got %#v", args)
		}
	})

	t.Run("unknown_falls_back_to_v6", func(t *testing.T) {
		builder := NewCommandBuilder("")
		if _, ok := builder.(*commandBuilderV6); !ok {
			t.Fatalf("NewCommandBuilder(empty version) returned %T, want *commandBuilderV6", builder)
		}
	})

	t.Run("v7_versions", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			minor := rapid.IntRange(0, 99).Draw(t, "minor")
			patch := rapid.IntRange(0, 99).Draw(t, "patch")
			version := fmt.Sprintf("7.%d.%d", minor, patch)

			builder := NewCommandBuilder(version)

			if !domain.IsRouterOSv7(version) {
				t.Errorf("IsRouterOSv7(%q) = false, want true", version)
			}

			if _, ok := builder.(*commandBuilderV7); !ok {
				t.Errorf("NewCommandBuilder(%q) returned %T, want *commandBuilderV7", version, builder)
			}

			params := domain.PPPoESecretParams{
				Name:     "testuser",
				Password: "testpass",
				Service:  "pppoe",
				Profile:  "default",
				Comment:  "ISPBoss:test:tenant",
			}
			command, _ := builder.CreateSecret(params)
			if command != "/ppp/secret/add" {
				t.Errorf("v7 CreateSecret command = %q, want /ppp/secret/add", command)
			}
		})
	})

	t.Run("v6_and_other_versions", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			prefix := rapid.SampledFrom([]string{"6", "5", "4", "3"}).Draw(t, "prefix")
			minor := rapid.IntRange(0, 99).Draw(t, "minor")
			patch := rapid.IntRange(0, 99).Draw(t, "patch")
			version := fmt.Sprintf("%s.%d.%d", prefix, minor, patch)

			builder := NewCommandBuilder(version)

			if domain.IsRouterOSv7(version) {
				t.Errorf("IsRouterOSv7(%q) = true, want false", version)
			}

			if _, ok := builder.(*commandBuilderV6); !ok {
				t.Errorf("NewCommandBuilder(%q) returned %T, want *commandBuilderV6", version, builder)
			}

			params := domain.PPPoESecretParams{
				Name:     "testuser",
				Password: "testpass",
				Service:  "pppoe",
				Profile:  "default",
				Comment:  "ISPBoss:test:tenant",
			}
			command, _ := builder.CreateSecret(params)
			if command != "/ppp/secret/add" {
				t.Errorf("v6 CreateSecret command = %q, want /ppp/secret/add", command)
			}
		})
	})
}

// =============================================================================
// =============================================================================

// customer_id, sumber IP, dan isolir method:
//
// **Memvalidasi: Kebutuhan 14.2, 14.3**
func TestProperty_IsolirNATRuleBuilderCorrectness(t *testing.T) {
	builders := []struct {
		name    string
		builder domain.CommandBuilder
	}{
		{"v6", NewCommandBuilder("6.49.10")},
		{"v7", NewCommandBuilder("7.14.3")},
	}

	for _, b := range builders {
		t.Run(b.name, func(t *testing.T) {
			builder := b.builder

			t.Run("firewall_nat_redirect", func(t *testing.T) {
				rapid.Check(t, func(t *rapid.T) {
					customerID := nonEmptyAlphanumString().Draw(t, "customerID")
					srcIP := ipv4String().Draw(t, "srcIP")
					toAddress := ipv4String().Draw(t, "toAddress")

					params := domain.NATRuleParams{
						Chain:      "dstnat",
						SrcAddress: srcIP,
						Protocol:   "tcp",
						DstPort:    "80",
						Action:     "dst-nat",
						ToAddress:  toAddress,
						Comment:    fmt.Sprintf("ISPBoss:isolir:%s", customerID),
					}

					command, args := builder.CreateNATRule(params)

					if command != "/ip/firewall/nat/add" {
						t.Errorf("command = %q, want /ip/firewall/nat/add", command)
					}

					if args["=chain"] != "dstnat" {
						t.Errorf("args[=chain] = %q, want dstnat", args["=chain"])
					}

					if args["=protocol"] != "tcp" {
						t.Errorf("args[=protocol] = %q, want tcp", args["=protocol"])
					}

					if args["=dst-port"] != "80" {
						t.Errorf("args[=dst-port] = %q, want 80", args["=dst-port"])
					}

					if args["=action"] != "dst-nat" {
						t.Errorf("args[=action] = %q, want dst-nat", args["=action"])
					}

					expectedComment := fmt.Sprintf("ISPBoss:isolir:%s", customerID)
					if args["=comment"] != expectedComment {
						t.Errorf("args[=comment] = %q, want %q", args["=comment"], expectedComment)
					}

					if args["=src-address"] != srcIP {
						t.Errorf("args[=src-address] = %q, want %q", args["=src-address"], srcIP)
					}
				})
			})

			t.Run("dns_redirect", func(t *testing.T) {
				rapid.Check(t, func(t *rapid.T) {
					customerID := nonEmptyAlphanumString().Draw(t, "customerID")
					srcIP := ipv4String().Draw(t, "srcIP")
					dnsServerIP := ipv4String().Draw(t, "dnsServerIP")

					params := domain.NATRuleParams{
						Chain:      "dstnat",
						SrcAddress: srcIP,
						Protocol:   "udp",
						DstPort:    "53",
						Action:     "dst-nat",
						ToAddress:  dnsServerIP,
						Comment:    fmt.Sprintf("ISPBoss:dns-redirect:%s", customerID),
					}

					command, args := builder.CreateNATRule(params)

					if command != "/ip/firewall/nat/add" {
						t.Errorf("command = %q, want /ip/firewall/nat/add", command)
					}

					if args["=chain"] != "dstnat" {
						t.Errorf("args[=chain] = %q, want dstnat", args["=chain"])
					}

					if args["=protocol"] != "udp" {
						t.Errorf("args[=protocol] = %q, want udp", args["=protocol"])
					}

					if args["=dst-port"] != "53" {
						t.Errorf("args[=dst-port] = %q, want 53", args["=dst-port"])
					}

					if args["=action"] != "dst-nat" {
						t.Errorf("args[=action] = %q, want dst-nat", args["=action"])
					}

					expectedComment := fmt.Sprintf("ISPBoss:dns-redirect:%s", customerID)
					if args["=comment"] != expectedComment {
						t.Errorf("args[=comment] = %q, want %q", args["=comment"], expectedComment)
					}

					if args["=src-address"] != srcIP {
						t.Errorf("args[=src-address] = %q, want %q", args["=src-address"], srcIP)
					}

					if !strings.Contains(args["=comment"], customerID) {
						t.Errorf("comment %q does not contain customerID %q", args["=comment"], customerID)
					}
				})
			})
		})
	}
}
