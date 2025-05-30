package types

// Oracle module event type constants
const (
	// EventTypeRegisterOracleRequestDoc defines the event type for registering oracle request document
	EventTypeRegisterOracleRequestDoc = "register_oracle_request_doc"

	// EventTypeUpdateOracleRequestDoc defines the event type for updating oracle request document
	EventTypeUpdateOracleRequestDoc = "update_oracle_request_doc"

	// EventTypeCompleteOracleDataSet defines the event type for complete oracle data set
	EventTypeCompleteOracleDataSet = "complete_oracle_data_set"
)

// Event attribute keys
const (
	AttributeKeyRequestID     = "request_id"
	AttributeKeyOracleType    = "oracle_type"
	AttributeKeyName          = "name"
	AttributeKeyDescription   = "description"
	AttributeKeyPeriod        = "period"
	AttributeKeyNodeList      = "node_list"
	AttributeKeyURLs          = "urls"
	AttributeKeyParseRule     = "parse_rule"
	AttributeKeyAggregateRule = "aggregate_rule"
	AttributeKeyStatus        = "status"
	AttributeKeyCreator       = "creator"
)

const (
	AttributeKeyOracleDataNone = "oracle_data_nonce"
)
