# Oracle Module

## Overview
The Oracle module is designed to handle oracle data requests and submissions in the Guru blockchain. It provides functionality for registering, updating, and managing oracle request documents, as well as submitting and querying oracle data.

## Genesis State

The Oracle module's genesis state contains the following parameters:

```json
{
  "oracle": {
    "params": {
      "enable_oracle": true,
      "submit_window": 3600,
      "min_submit_per_window": "0.5",
      "slash_fraction_downtime": "0.01"
    },
    "oracle_request_docs": [],
    "moderator_address": "guru1..."
  }
}
```

### Parameters
- `enable_oracle`: Whether the oracle module is enabled
- `submit_window`: The window within which oracle data is expected to be submitted
- `min_submit_per_window`: Minimum number of submissions required per window (as a decimal)
- `slash_fraction_downtime`: Fraction of stake to slash for downtime (as a decimal)

### Export Genesis State
To export the current state of the Oracle module:

```bash
gurud export --home <path-to-home> | jq '.app_state.oracle'
```

### Import Genesis State
To import a genesis state for the Oracle module:

```bash
# Create a new genesis file with oracle state
jq '.app_state.oracle = {
  "params": {
    "enable_oracle": true,
    "submit_window": 3600,
    "min_submit_per_window": "0.5",
    "slash_fraction_downtime": "0.01"
  },
  "oracle_request_docs": [],
  "moderator_address": "guru1..."
}' genesis.json > new-genesis.json

# Replace the old genesis file
mv new-genesis.json genesis.json
```

## Transactions

### Register Oracle Request Document
Register a new oracle request document.
```bash
gurud tx oracle register-request [path/to/request-doc.json]
```

### Update Oracle Request Document
Update an existing oracle request document.
```bash
gurud tx oracle update-request [path/to/request-doc.json] [reason]
```

### Submit Oracle Data
Submit oracle data for a specific request.
```bash
gurud tx oracle submit-data [request-id] [nonce] [raw-data]
```

### Update Moderator Address
Update the moderator address for the oracle module.
```bash
gurud tx oracle update-moderator-address [moderator-address]
```

## Queries

### Parameters
Query the current oracle parameters.
```bash
gurud query oracle params
```

### Oracle Request Document
Query a specific oracle request document by ID.
```bash
gurud query oracle request-doc [request-id]
```

### Oracle Data
Query oracle data for a specific request.
```bash
gurud query oracle data [request-id]
```

### Oracle Submit Data
Query oracle submit data for a request.
```bash
gurud query oracle submit-data [request-id] [nonce] [provider-account]
```

### Oracle Request Documents
Query all oracle request documents.
```bash
gurud query oracle request-docs
```

### Moderator Address
Query the current moderator address.
```bash
gurud query oracle moderator-address
```

## CLI Examples

### Register a New Oracle Request
```bash
# Create a request document JSON file
# request_id is omitted.
cat > request.json << EOF
{
  "oracle_type": 1,
  "name": "BTC/USD Price Oracle",
  "description": "Provides real-time BTC/USD price data from multiple sources",
  "period": 60,
  "account_list": [
    "guru1...",
    "guru1..."
  ],
  "quorum": 3,
  "urls": [
    "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd",
    "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT"
  ],
  "parse_rule": "$.bitcoin.usd",
  "aggregation_rule": 1,
  "status": 1,
}
EOF

# Register the request
gurud tx oracle register-request request.json --from mykey
```

### Update an Oracle Request
```bash
# Create an updated request document JSON file
# It is mandatory to include the request_id. Only [period, status, account_list, quorum, urls, parser_rule, aggregation_rule] can be updated. Remove any items that do not need to be updated.
cat > updated_request.json << EOF
{
  "request_id": 1,
  "period": 30,
  "account_list": [
    "guru1...",
    "guru1...",
    "guru1..."
  ],
  "quorum": 4,
  "urls": [
    "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd",
    "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT",
    "https://api.kraken.com/0/public/Ticker?pair=XBTUSD"
  ],
  "parse_rule": "$.bitcoin.usd",
  "aggregation_rule": 4,
  "status": 1,
}
EOF

# Update the request with a reason
gurud tx oracle update-request updated_request.json "Improving data reliability and update frequency" --from mykey
```

### Submit Oracle Data
```bash
# Submit data for request ID 1 with nonce 1
gurud tx oracle submit-data 1 1 100 --from mykey
```

### Query Oracle Data
```bash
# Query data for request ID 1
gurud query oracle data 1
```

### Update Moderator
```bash
# Update moderator address
gurud tx oracle update-moderator-address guru1... --from current-moderator-address
``` 