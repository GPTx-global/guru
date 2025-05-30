package types

import (
	"fmt"
)

// type RequestDocument struct {
// 	RequestOracleDoc
// 	nonce uint64
// }

// Validate performs basic validation on RequestOracleDoc
func (doc RequestOracleDoc) Validate() error {
	// Check if oracle type is unspecified
	if doc.OracleType == OracleType_ORACLE_TYPE_UNSPECIFIED {
		return fmt.Errorf("oracle type cannot be unspecified")
	}
	// Check if oracle type is zero (empty)
	if doc.OracleType == 0 {
		return fmt.Errorf("oracle type cannot be empty")
	}

	return nil
}
