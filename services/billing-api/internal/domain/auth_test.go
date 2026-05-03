package domain

import (
	"encoding/json"
	"testing"

	"pgregory.net/rapid"
)

// Feature: auth-rbac, Property 18: API Response Format Consistency
// **Validates: Requirements 16.1, 16.2**
//
// For any API response from auth endpoints, successful responses SHALL have the
// structure {"success": true, "data": {...}} and error responses SHALL have the
// structure {"success": false, "error": {"code": "...", "message": "..."}}.
// No response SHALL contain both data and error fields simultaneously.
func TestProperty_APIResponseFormatConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		isSuccess := rapid.Bool().Draw(t, "isSuccess")

		var resp APIResponse

		if isSuccess {
			// Build a success response: Success=true, Data is non-nil, Error is nil
			data := map[string]interface{}{
				"id":   rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`).Draw(t, "data_id"),
				"name": rapid.StringMatching(`[a-zA-Z ]{3,50}`).Draw(t, "data_name"),
			}
			resp = APIResponse{
				Success: true,
				Data:    data,
				Error:   nil,
			}
		} else {
			// Build an error response: Success=false, Error is non-nil, Data is nil
			errorCodes := []string{
				"VALIDATION_ERROR", "UNAUTHORIZED", "INVALID_CREDENTIALS",
				"FORBIDDEN", "EMAIL_NOT_VERIFIED", "ACCOUNT_DISABLED",
				"TOKEN_EXPIRED", "TOKEN_ALREADY_USED", "TOKEN_NOT_FOUND",
				"EMAIL_ALREADY_EXISTS", "ACCOUNT_LOCKED", "INTERNAL_ERROR",
			}
			code := rapid.SampledFrom(errorCodes).Draw(t, "error_code")
			message := rapid.StringMatching(`[a-zA-Z ]{5,100}`).Draw(t, "error_message")

			// Optionally include field-level details
			includeDetails := rapid.Bool().Draw(t, "include_details")
			var details []FieldError
			if includeDetails {
				numDetails := rapid.IntRange(1, 5).Draw(t, "num_details")
				for i := 0; i < numDetails; i++ {
					details = append(details, FieldError{
						Field:   rapid.SampledFrom([]string{"name", "email", "phone", "password", "role"}).Draw(t, "field"),
						Message: rapid.StringMatching(`[a-zA-Z ]{5,50}`).Draw(t, "field_message"),
					})
				}
			}

			resp = APIResponse{
				Success: false,
				Error: &APIError{
					Code:    code,
					Message: message,
					Details: details,
				},
				Data: nil,
			}
		}

		// Marshal to JSON
		jsonBytes, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}

		// Unmarshal into a generic map to inspect the raw JSON structure
		var raw map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &raw); err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
		}

		// Property: "success" field must always be present
		successVal, ok := raw["success"]
		if !ok {
			t.Fatal("JSON response missing 'success' field")
		}

		successBool, ok := successVal.(bool)
		if !ok {
			t.Fatalf("'success' field is not a boolean: %T", successVal)
		}

		if isSuccess {
			// Success responses: success must be true
			if !successBool {
				t.Error("Success response has 'success': false")
			}

			// Success responses must have "data" key with non-null value
			dataVal, hasData := raw["data"]
			if !hasData {
				t.Error("Success response missing 'data' field")
			}
			if dataVal == nil {
				t.Error("Success response has null 'data' field")
			}

			// Success responses must NOT have "error" key with non-null value
			errorVal, hasError := raw["error"]
			if hasError && errorVal != nil {
				t.Error("Success response has non-null 'error' field — data and error must not coexist")
			}
		} else {
			// Error responses: success must be false
			if successBool {
				t.Error("Error response has 'success': true")
			}

			// Error responses must have "error" key with non-null value
			errorVal, hasError := raw["error"]
			if !hasError {
				t.Error("Error response missing 'error' field")
			}
			if errorVal == nil {
				t.Error("Error response has null 'error' field")
			}

			// Verify error structure has code and message
			if errorMap, ok := errorVal.(map[string]interface{}); ok {
				if _, hasCode := errorMap["code"]; !hasCode {
					t.Error("Error response missing 'error.code' field")
				}
				if _, hasMessage := errorMap["message"]; !hasMessage {
					t.Error("Error response missing 'error.message' field")
				}
			}

			// Error responses must NOT have "data" key with non-null value
			dataVal, hasData := raw["data"]
			if hasData && dataVal != nil {
				t.Error("Error response has non-null 'data' field — data and error must not coexist")
			}
		}

		// Cross-check: unmarshal back into APIResponse and verify consistency
		var roundTrip APIResponse
		if err := json.Unmarshal(jsonBytes, &roundTrip); err != nil {
			t.Fatalf("Round-trip unmarshal failed: %v", err)
		}

		if roundTrip.Success != resp.Success {
			t.Errorf("Round-trip Success mismatch: got %v, want %v", roundTrip.Success, resp.Success)
		}
	})
}
