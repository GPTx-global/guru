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

	return ValidateExchangeRequiredKeys(exchange)
}

func ValidateExchangeRequiredKeys(exchange *Exchange) error {
	size := len(RequiredKeysExchange)
	count := 0

	for _, attribute := range exchange.Attributes {
		for _, key := range RequiredKeysExchange {
			if attribute.Key == *key {
				count++
				break
			}
		}
	}

	keys := ""
	for _, key := range RequiredKeysExchange {
		keys += *key + ", "
	}

	if count < size {
		return errorsmod.Wrapf(ErrRequiredKey, "required keys: %s", keys)
	}

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

func AttributesToMap(attributes []Attribute) map[string]string {
	attributesMap := make(map[string]string)
	for _, attribute := range attributes {
		attributesMap[attribute.Key] = attribute.Value
	}
	return attributesMap
}
