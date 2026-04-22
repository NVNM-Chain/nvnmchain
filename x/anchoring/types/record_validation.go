package types

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	MaxRecordChecksumLen     = 128
	MaxRecordChecksumAlgoLen = 128
	MaxRecordURILen          = 2048
	MaxRecordMetadataLen     = 2048
	MaxRecordStatusLen       = 64
)

var hexPattern = regexp.MustCompile(`^[a-fA-F0-9]+$`)

// knownAlgos maps canonical algorithm names (lowercase, hyphens stripped) to their expected hex checksum length.
var knownAlgos = map[string]int{
	"sha256":  64,
	"sha512":  128,
	"sha3256": 64,
}

func validateMaxLen(field string, value string, max int) error {
	if max <= 0 {
		return fmt.Errorf("invalid max length for %s", field)
	}
	if len(value) > max {
		return fmt.Errorf("%s exceeds max length: got=%d max=%d", field, len(value), max)
	}
	return nil
}

// ValidateRecordForCreate validates a record for creation.
func ValidateRecordForCreate(record Record) error {
	if record.Checksum == "" {
		return fmt.Errorf("checksum cannot be empty")
	}
	if record.ChecksumAlgo == "" {
		return fmt.Errorf("checksum algorithm cannot be empty")
	}
	if record.Uri == "" {
		return fmt.Errorf("uri cannot be empty")
	}
	if record.Metadata == "" || record.Metadata == "{}" {
		return fmt.Errorf("metadata cannot be empty")
	}
	if record.Registry == "" {
		return fmt.Errorf("registry cannot be empty")
	}

	if err := validateMaxLen("checksum", record.Checksum, MaxRecordChecksumLen); err != nil {
		return err
	}
	if err := validateMaxLen("checksum algorithm", record.ChecksumAlgo, MaxRecordChecksumAlgoLen); err != nil {
		return err
	}
	if err := validateMaxLen("uri", record.Uri, MaxRecordURILen); err != nil {
		return err
	}
	if err := validateMaxLen("metadata", record.Metadata, MaxRecordMetadataLen); err != nil {
		return err
	}
	if err := ValidateRecordStatus(record.Status); err != nil {
		return err
	}
	if err := validateChecksum(record.ChecksumAlgo, record.Checksum); err != nil {
		return err
	}

	return nil
}

func validateChecksum(algo, checksum string) error {
	normalized := strings.ReplaceAll(strings.ToLower(algo), "-", "")
	expectedLen, ok := knownAlgos[normalized]
	if !ok {
		return fmt.Errorf("unsupported checksum algorithm %q: supported values are sha256 (alias sha-256), sha512 (alias sha-512), and sha3-256; matching is case-insensitive", algo)
	}
	if !hexPattern.MatchString(checksum) {
		return fmt.Errorf("checksum must be hex-encoded (^[a-fA-F0-9]+$)")
	}
	if len(checksum) != expectedLen {
		return fmt.Errorf("checksum length %d does not match algorithm %s (expected %d hex characters)", len(checksum), algo, expectedLen)
	}
	return nil
}

func ValidateRecordStatus(status string) error {
	if status == "" {
		return fmt.Errorf("status cannot be empty")
	}
	return validateMaxLen("status", status, MaxRecordStatusLen)
}
