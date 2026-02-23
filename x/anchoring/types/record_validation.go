package types

import "fmt"

const (
	MaxRecordChecksumLen     = 256
	MaxRecordChecksumAlgoLen = 128
	MaxRecordURILen          = 2048
	MaxRecordMetadataLen     = 2048
	MaxRecordStatusLen       = 64
)

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
	if record.Status == "" {
		return fmt.Errorf("status cannot be empty")
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
	if err := validateMaxLen("status", record.Status, MaxRecordStatusLen); err != nil {
		return err
	}

	return nil
}

func ValidateRecordStatus(status string) error {
	if status == "" {
		return fmt.Errorf("status cannot be empty")
	}
	return validateMaxLen("status", status, MaxRecordStatusLen)
}
