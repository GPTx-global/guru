package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
)

func ValidateExchange(exchange *Exchange) error {
	if exchange.Id.IsNil() || exchange.Id.IsZero() {
		return errorsmod.Wrapf(ErrInvalidExchangeId, "zero or nil id not allowed")
	}

	for _, attribute := range exchange.Attributes {
		if err := ValidateAttribute(attribute); err != nil {
			return err
		}
	}
	return nil
}

func ValidateExchangeRequiredKeys(exchange *Exchange) error {
	return nil
}

func ValidateAttribute(attribute Attribute) error {
	if attribute.Key == "" {
		return fmt.Errorf("key is empty")
	}

	if attribute.Value == "" {
		return fmt.Errorf("value is empty")
	}

	return nil
}
