package types

// distribution module event types
const (
	EventTypeAddRegistry        = "add_registry"
	EventTypeAddRecord          = "add_record"
	EventTypeUpdateRecordStatus = "update_record_status"
	EventTypeGrantRole          = "grant_role"
	EventTypeRevokeRole         = "revoke_role"

	// attribute keys
	AttributeKeyRegistryName   = "registry_name"
	AttributeKeyRegistryId     = "registry_id"
	AttributeKeyRecordId       = "record_id"
	AttributeKeyRecordChecksum = "record_checksum"
	AttributeKeyRecordIndex    = "record_index"
	AttributeKeyRecordStatus   = "record_status"
	AttributeKeyGrantee        = "grantee"
	AttributeKeyGranter        = "granter"
	AttributeKeyRevokee        = "revokee"
	AttributeKeyRevoker        = "revoker"
	AttributeKeyRole           = "role"
)
