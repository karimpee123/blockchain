package solprogram

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ProgramErrors codes from Rust
var ProgramErrors = map[int]string{
	6000: "InvalidOwner - You are not the owner of this envelope",
	6001: "AlreadyClaimed - You have already claimed this envelope",
	6002: "NotAllowed - You are not allowed to claim this envelope",
	6003: "QuotaFull - Maximum claimers reached",
	6004: "Expired - Envelope has expired",
	6005: "NotExpired - Envelope not expired yet (cannot refund)",
	6006: "ExceedMaxCreate - Amount exceeds maximum allowed (10 SOL)",
	6007: "NotExpired - Envelope not expired yet",
	6008: "MathOverflow - Math calculation overflow",
	6009: "InsufficientFunds - Insufficient funds in envelope",
	6010: "NothingToRefund - Nothing to refund",
}

// ExtractErrorCode tries multiple methods to extract custom program error code
func ExtractErrorCode(err error) *int {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Method 1: Try to parse JSON structure
	// Format: "err": {"InstructionError": [0, {"Custom": 6002}]}
	type CustomError struct {
		Custom interface{} `json:"Custom"`
	}
	type InstructionErrorData struct {
		InstructionError []interface{} `json:"InstructionError"`
	}
	type ErrorWrapper struct {
		Err InstructionErrorData `json:"err"`
	}

	// Find JSON portion in error string
	if jsonStart := strings.Index(errStr, `"err":`); jsonStart != -1 {
		// Extract balanced JSON object
		jsonStr := errStr[jsonStart-1:]
		braceCount := 0
		endPos := -1

		for i, ch := range jsonStr {
			if ch == '{' {
				braceCount++
			} else if ch == '}' {
				braceCount--
				if braceCount == 0 {
					endPos = i + 1
					break
				}
			}
		}

		if endPos > 0 {
			jsonStr = "{" + jsonStr[:endPos]

			var wrapper ErrorWrapper
			if err := json.Unmarshal([]byte(jsonStr), &wrapper); err == nil {
				if len(wrapper.Err.InstructionError) >= 2 {
					if customMap, ok := wrapper.Err.InstructionError[1].(map[string]interface{}); ok {
						if customVal, ok := customMap["Custom"]; ok {
							// Handle different JSON number types
							switch v := customVal.(type) {
							case float64:
								code := int(v)
								return &code
							case string:
								if code, err := strconv.Atoi(v); err == nil {
									return &code
								}
							}
						}
					}
				}
			}
		}
	}

	// Method 2: Regex patterns for "Custom": 6002
	patterns := []string{
		`"Custom":\s*(\d+)`,     // "Custom": 6002
		`"Custom":\s*"(\d+)"`,   // "Custom": "6002"
		`Custom:\s*(\d+)`,       // Custom: 6002
		`error code:\s*(\d+)`,   // error code: 6002
		`Error Number:\s*(\d+)`, // Error Number: 6002 (from Anchor logs)
	}

	for _, pattern := range patterns {
		if matches := regexp.MustCompile(pattern).FindStringSubmatch(errStr); len(matches) > 1 {
			if code, err := strconv.Atoi(matches[1]); err == nil {
				return &code
			}
		}
	}

	// Method 3: Hex format - custom program error: 0x1772
	if matches := regexp.MustCompile(`custom program error: 0x([0-9a-fA-F]+)`).FindStringSubmatch(errStr); len(matches) > 1 {
		if code, err := strconv.ParseInt(matches[1], 16, 64); err == nil {
			intCode := int(code)
			return &intCode
		}
	}

	return nil
}

// ParseSolanaError extracts and formats error
func ParseSolanaError(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	// Check for BlockhashNotFound (transaction expired)
	if strings.Contains(errStr, "BlockhashNotFound") ||
		strings.Contains(errStr, "Blockhash not found") {
		return "Transaction expired. The blockhash is no longer valid. Please create a new transaction and try again."
	}

	// Try to get custom program error code
	if code := ExtractErrorCode(err); code != nil {
		if msg, ok := ProgramErrors[*code]; ok {
			return msg
		}
		return fmt.Sprintf("Custom program error code: %d", *code)
	}

	// Check for simulation failed
	if regexp.MustCompile(`simulation failed`).MatchString(errStr) {
		return "Transaction simulation failed. Check program logs for details."
	}

	// Check for insufficient funds
	if regexp.MustCompile(`insufficient funds`).MatchString(errStr) {
		return "Insufficient SOL balance to pay for transaction"
	}

	// Return truncated error
	if len(errStr) > 300 {
		return errStr[:300] + "..."
	}
	return errStr
}

// ExtractLogMessages extracts program logs from error
func ExtractLogMessages(err error) []string {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	logs := []string{}

	// Pattern to extract "Program log: ..." messages
	// Handle both escaped and non-escaped strings
	patterns := []string{
		`Program log: ([^"\\n]+?)(?:"|\\n|$)`, // With quotes
		`Program log: ([^\n]+)`,               // Without quotes
	}

	for _, pattern := range patterns {
		matches := regexp.MustCompile(pattern).FindAllStringSubmatch(errStr, -1)
		for _, match := range matches {
			if len(match) > 1 {
				log := strings.TrimSpace(match[1])
				if log != "" && !contains(logs, log) {
					logs = append(logs, log)
				}
			}
		}
	}

	return logs
}

// Helper function to check if slice contains string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
