package usecase

import (
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// Generators — reusable generators untuk property tests
// =============================================================================

// uuidGen menghasilkan UUID v4 string acak.
func uuidGen() *rapid.Generator[string] {
	return rapid.StringMatching(
		`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`,
	)
}

// pppoeUsernameGen menghasilkan username PPPoE yang valid (alfanumerik + hyphen/underscore).
func pppoeUsernameGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z][a-z0-9_\-]{0,19}`)
}

// profileNameGen menghasilkan profile name yang valid.
func profileNameGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z][a-z0-9\-]{0,19}`)
}

// dbUserGen menghasilkan domain.PPPoEUser acak untuk testing.
func dbUserGen(username string) *rapid.Generator[*domain.PPPoEUser] {
	return rapid.Custom[*domain.PPPoEUser](func(t *rapid.T) *domain.PPPoEUser {
		customerID := uuidGen().Draw(t, "customerID")
		tenantID := uuidGen().Draw(t, "tenantID")
		return &domain.PPPoEUser{
			ID:          uuidGen().Draw(t, "id"),
			TenantID:    tenantID,
			CustomerID:  customerID,
			RouterID:    uuidGen().Draw(t, "routerID"),
			Username:    username,
			ProfileName: profileNameGen().Draw(t, "profileName"),
			Service:     "pppoe",
			Comment:     domain.BuildComment(customerID, tenantID),
			Disabled:    rapid.Bool().Draw(t, "disabled"),
			Status:      "active",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	})
}

// routerSecretGen menghasilkan router secret map dengan comment ISPBoss.
func routerSecretGen(username, profile string, disabled bool) map[string]string {
	disabledStr := "false"
	if disabled {
		disabledStr = "true"
	}
	return map[string]string{
		"name":     username,
		"profile":  profile,
		"disabled": disabledStr,
		"comment":  domain.BuildComment("cust-123", "tenant-456"),
	}
}

// orphanSecretGen menghasilkan router secret tanpa comment ISPBoss (orphan manual).
func orphanSecretGen(t *rapid.T) map[string]string {
	return map[string]string{
		"name":     pppoeUsernameGen().Draw(t, "orphanUsername"),
		"profile":  profileNameGen().Draw(t, "orphanProfile"),
		"disabled": "false",
		"comment":  rapid.StringMatching(`[a-zA-Z0-9 ]{0,20}`).Draw(t, "orphanComment"),
	}
}

// =============================================================================
// Feature: mikrotik-pppoe, Property 5: Sync diff algorithm correctness
// =============================================================================

// TestProperty_SyncDiffAlgorithmCorrectness memverifikasi bahwa untuk sembarang
// set router PPPoE users dan database PPPoE users, sync diff algorithm
// mengkategorikan setiap user ke dalam tepat satu kategori: synced, orphan,
// missing, atau out_of_sync. Setiap user dari kedua set muncul di tepat satu
// kategori (tidak ada duplikat, tidak ada yang terlewat).
//
// **Validates: Requirements 8.2, 8.3, 8.5**
func TestProperty_SyncDiffAlgorithmCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a set of unique usernames for DB users
		numDBUsers := rapid.IntRange(0, 15).Draw(t, "numDBUsers")
		usernameSet := make(map[string]bool)
		var dbUsers []*domain.PPPoEUser

		for i := 0; i < numDBUsers; i++ {
			var username string
			for {
				username = pppoeUsernameGen().Draw(t, "dbUsername")
				if !usernameSet[username] {
					break
				}
			}
			usernameSet[username] = true
			user := dbUserGen(username).Draw(t, "dbUser")
			dbUsers = append(dbUsers, user)
		}

		// Generate router secrets: mix of synced, out_of_sync, orphan (ISPBoss), orphan (non-ISPBoss)
		var routerSecrets []map[string]string

		// Category 1: Synced users — same username, same profile, same disabled
		numSynced := rapid.IntRange(0, len(dbUsers)).Draw(t, "numSynced")
		syncedUsers := dbUsers[:numSynced]
		for _, u := range syncedUsers {
			routerSecrets = append(routerSecrets, routerSecretGen(u.Username, u.ProfileName, u.Disabled))
		}

		// Category 2: Out-of-sync users — same username, different profile or disabled
		numOutOfSync := rapid.IntRange(0, len(dbUsers)-numSynced).Draw(t, "numOutOfSync")
		outOfSyncUsers := dbUsers[numSynced : numSynced+numOutOfSync]
		for _, u := range outOfSyncUsers {
			// Flip either profile or disabled to make it out of sync
			flipProfile := rapid.Bool().Draw(t, "flipProfile")
			profile := u.ProfileName
			disabled := u.Disabled
			if flipProfile {
				profile = profileNameGen().Draw(t, "differentProfile")
				// Ensure it's actually different
				if profile == u.ProfileName {
					profile = profile + "x"
				}
			} else {
				disabled = !u.Disabled
			}
			routerSecrets = append(routerSecrets, routerSecretGen(u.Username, profile, disabled))
		}

		// Remaining DB users (not on router) will be "missing"
		expectedMissing := len(dbUsers) - numSynced - numOutOfSync

		// Category 3: Orphan users with ISPBoss comment but not in DB
		numOrphanISPBoss := rapid.IntRange(0, 5).Draw(t, "numOrphanISPBoss")
		for i := 0; i < numOrphanISPBoss; i++ {
			var orphanUsername string
			for {
				orphanUsername = pppoeUsernameGen().Draw(t, "orphanISPBossUsername")
				if !usernameSet[orphanUsername] {
					break
				}
			}
			usernameSet[orphanUsername] = true
			routerSecrets = append(routerSecrets, map[string]string{
				"name":     orphanUsername,
				"profile":  profileNameGen().Draw(t, "orphanISPBossProfile"),
				"disabled": "false",
				"comment":  domain.BuildComment("orphan-cust", "orphan-tenant"),
			})
		}

		// Category 4: Orphan users without ISPBoss comment (manual admin users)
		numOrphanManual := rapid.IntRange(0, 5).Draw(t, "numOrphanManual")
		for i := 0; i < numOrphanManual; i++ {
			routerSecrets = append(routerSecrets, orphanSecretGen(t))
		}

		expectedOrphans := numOrphanISPBoss + numOrphanManual

		// Run the diff
		diff := computeSyncDiff(routerSecrets, dbUsers)

		// Property: every user appears in exactly one category
		totalCategorized := diff.SyncedCount + diff.OrphanCount + diff.MissingCount + diff.OutOfSyncCount

		// Total categorized should equal:
		// - all router secrets (each goes to synced, orphan, or out_of_sync)
		// - plus missing DB users (active, not deleted, not on router)
		expectedTotal := len(routerSecrets) + expectedMissing
		if totalCategorized != expectedTotal {
			t.Errorf("total categorized=%d, expected=%d (routerSecrets=%d + missing=%d); synced=%d, orphan=%d, missing=%d, out_of_sync=%d",
				totalCategorized, expectedTotal, len(routerSecrets), expectedMissing,
				diff.SyncedCount, diff.OrphanCount, diff.MissingCount, diff.OutOfSyncCount)
		}

		// Property: counts match expected categories
		if diff.SyncedCount != numSynced {
			t.Errorf("synced=%d, expected=%d", diff.SyncedCount, numSynced)
		}
		if diff.OutOfSyncCount != numOutOfSync {
			t.Errorf("out_of_sync=%d, expected=%d", diff.OutOfSyncCount, numOutOfSync)
		}
		if diff.OrphanCount != expectedOrphans {
			t.Errorf("orphan=%d, expected=%d", diff.OrphanCount, expectedOrphans)
		}
		if diff.MissingCount != expectedMissing {
			t.Errorf("missing=%d, expected=%d", diff.MissingCount, expectedMissing)
		}
	})
}

// =============================================================================
// Feature: mikrotik-pppoe, Property 6: Sync result count invariant
// =============================================================================

// TestProperty_SyncResultCountInvariant memverifikasi bahwa untuk sembarang
// SyncResult, total_users >= synced_count + missing_count + out_of_sync_count + error_count.
// orphan_count merepresentasikan user yang hanya ada di router.
//
// **Validates: Requirements 8.7**
func TestProperty_SyncResultCountInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate DB users and router secrets, then compute diff and build SyncResult
		numDBUsers := rapid.IntRange(0, 20).Draw(t, "numDBUsers")
		usernameSet := make(map[string]bool)
		var dbUsers []*domain.PPPoEUser

		for i := 0; i < numDBUsers; i++ {
			var username string
			for {
				username = pppoeUsernameGen().Draw(t, "username")
				if !usernameSet[username] {
					break
				}
			}
			usernameSet[username] = true
			user := dbUserGen(username).Draw(t, "user")
			dbUsers = append(dbUsers, user)
		}

		// Generate router secrets: some matching DB users, some orphans
		var routerSecrets []map[string]string

		// Add some DB users to router (synced or out-of-sync)
		numOnRouter := rapid.IntRange(0, numDBUsers).Draw(t, "numOnRouter")
		for i := 0; i < numOnRouter; i++ {
			u := dbUsers[i]
			// Randomly make synced or out-of-sync
			if rapid.Bool().Draw(t, "isSynced") {
				routerSecrets = append(routerSecrets, routerSecretGen(u.Username, u.ProfileName, u.Disabled))
			} else {
				routerSecrets = append(routerSecrets, routerSecretGen(u.Username, u.ProfileName+"diff", !u.Disabled))
			}
		}

		// Add orphan secrets (not in DB)
		numOrphans := rapid.IntRange(0, 10).Draw(t, "numOrphans")
		for i := 0; i < numOrphans; i++ {
			var orphanUsername string
			for {
				orphanUsername = pppoeUsernameGen().Draw(t, "orphanUsername")
				if !usernameSet[orphanUsername] {
					break
				}
			}
			usernameSet[orphanUsername] = true
			routerSecrets = append(routerSecrets, map[string]string{
				"name":     orphanUsername,
				"profile":  "some-profile",
				"disabled": "false",
				"comment":  domain.BuildComment("orphan-cust", "orphan-tenant"),
			})
		}

		// Compute diff
		diff := computeSyncDiff(routerSecrets, dbUsers)

		// Build SyncResult as SyncRouter would
		result := domain.SyncResult{
			RouterID:       "test-router",
			TotalUsers:     len(dbUsers),
			SyncedCount:    diff.SyncedCount,
			OrphanCount:    diff.OrphanCount,
			MissingCount:   diff.MissingCount,
			OutOfSyncCount: diff.OutOfSyncCount,
			ErrorCount:     0, // computeSyncDiff doesn't produce errors (pure function)
			SyncedAt:       time.Now(),
		}

		// Property: total_users >= synced_count + missing_count + out_of_sync_count + error_count
		// This holds because total_users = len(dbUsers), and synced + missing + out_of_sync
		// accounts for all active DB users (some may be inactive/deleted and not counted)
		sumDBCategories := result.SyncedCount + result.MissingCount + result.OutOfSyncCount + result.ErrorCount
		if result.TotalUsers < sumDBCategories {
			t.Errorf("invariant violated: total_users(%d) < synced(%d) + missing(%d) + out_of_sync(%d) + error(%d) = %d",
				result.TotalUsers, result.SyncedCount, result.MissingCount,
				result.OutOfSyncCount, result.ErrorCount, sumDBCategories)
		}

		// Property: orphan_count represents users only on the router
		// total_users (DB) + orphan_count = total unique users across both sources
		// (minus non-ISPBoss orphans which are also counted)
		// The orphan count should be >= 0
		if result.OrphanCount < 0 {
			t.Errorf("orphan_count should be >= 0, got %d", result.OrphanCount)
		}

		// Property: synced + out_of_sync <= total_users (can't have more matched users than DB users)
		if result.SyncedCount+result.OutOfSyncCount > result.TotalUsers {
			t.Errorf("synced(%d) + out_of_sync(%d) > total_users(%d)",
				result.SyncedCount, result.OutOfSyncCount, result.TotalUsers)
		}

		// Property: all counts are non-negative
		if result.SyncedCount < 0 || result.OrphanCount < 0 || result.MissingCount < 0 ||
			result.OutOfSyncCount < 0 || result.ErrorCount < 0 {
			t.Errorf("negative count detected: synced=%d, orphan=%d, missing=%d, out_of_sync=%d, error=%d",
				result.SyncedCount, result.OrphanCount, result.MissingCount,
				result.OutOfSyncCount, result.ErrorCount)
		}
	})
}

// =============================================================================
// Unit Tests — parseDisabledField
// =============================================================================

// TestParseDisabledField memverifikasi bahwa parseDisabledField mengurai
// field "disabled" dari router response dengan benar.
func TestParseDisabledField(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"yes", true},
		{"Yes", true},
		{"YES", true},
		{"  true  ", true},
		{"  yes  ", true},
		{"false", false},
		{"False", false},
		{"no", false},
		{"No", false},
		{"", false},
		{"0", false},
		{"1", false},
		{"random", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDisabledField(tt.input)
			if result != tt.expected {
				t.Errorf("parseDisabledField(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
