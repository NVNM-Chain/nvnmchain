package types

import "fmt"

const (
	MaxRecordChecksumLen     = 64
	MaxRecordChecksumAlgoLen = 128
	MaxRecordURILen          = 2048
	MaxRecordMetadataLen     = 2048
	MaxRecordStatusLen       = 64

	ErrMsgRegistryIDZero = "registry ID cannot be zero"
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

	return nil
}

// ValidateRecord validates a Record for genesis.
func ValidateRecord(r Record) error {
	if r.RecordId <= 0 {
		return fmt.Errorf("record_id cannot be <= 0")
	}
	if r.Index <= 0 {
		return fmt.Errorf("index cannot be <= 0")
	}
	return ValidateRecordForCreate(r)
}

// RecordVersionGroup tracks IsLatest consistency across versions of a single record.
type RecordVersionGroup struct {
	MaxIndex    uint64
	LatestIndex uint64
	LatestCount int
}

func (g *RecordVersionGroup) Add(r Record) {
	if r.Index > g.MaxIndex {
		g.MaxIndex = r.Index
	}
	if r.IsLatest {
		g.LatestIndex = r.Index
		g.LatestCount++
	}
}

// ValidateIsLatest returns an error if exactly one record with IsLatest==true
// on the highest index is not present. label identifies the group in the error.
func (g *RecordVersionGroup) ValidateIsLatest(label string) error {
	if g.LatestCount == 0 {
		return fmt.Errorf("%s has no record with is_latest=true", label)
	}
	if g.LatestCount > 1 {
		return fmt.Errorf("%s has %d records with is_latest=true, expected 1", label, g.LatestCount)
	}
	if g.LatestIndex != g.MaxIndex {
		return fmt.Errorf("%s is_latest=true is on index %d but max index is %d", label, g.LatestIndex, g.MaxIndex)
	}
	return nil
}

func ValidateRecordStatus(status string) error {
	if status == "" {
		return fmt.Errorf("status cannot be empty")
	}
	return validateMaxLen("status", status, MaxRecordStatusLen)
}
