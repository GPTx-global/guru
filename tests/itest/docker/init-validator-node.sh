#!/bin/bash

source env.sh

GENESIS_PARAMS=("$@")

KEY="dev0"

# validate dependencies are installed
command -v jq > /dev/null 2>&1 || { echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"; exit 1; }

# used to exit on first error (any non-zero exit code)
set -e

rm -rf "$CHAINDIR"

# Set client config
$DNOM config keyring-backend "$KEYRING"
$DNOM config chain-id "$CHAINID"

# if $KEY exists it should be deleted
# $DNOM keys add "$KEY" --keyring-backend $KEYRING --algo "$KEYALGO" --home "$CHAINDIR"
echo "${MNICS[$START_INDEX_VAL]}" | $DNOM keys add "$KEY" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$CHAINDIR"
echo "${MNICS[0]}" | $DNOM keys add "$BASEKEY" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$CHAINDIR"
echo "${MNICS[1]}" | $DNOM keys add "$MODKEY" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$CHAINDIR"
echo "${MNICS[2]}" | $DNOM keys add "tester1" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$CHAINDIR"
echo "${MNICS[3]}" | $DNOM keys add "tester2" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$CHAINDIR"
echo "${MNICS[4]}" | $DNOM keys add "tester3" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$CHAINDIR"

# Set moniker and chain-id for Guru (Moniker can be anything, chain-id must be an integer)
$DNOM init "$MONIKER" --chain-id "$CHAINID"

# Change parameter token denominations to aguru
jq '.app_state.staking.params.bond_denom="aguru"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
jq '.app_state.crisis.constant_fee.denom="aguru"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
jq '.app_state.gov.deposit_params.min_deposit[0].denom="aguru"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
jq '.app_state.evm.params.evm_denom="aguru"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
jq '.app_state.inflation.params.mint_denom="aguru"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

# set gov proposing && voting period
jq '.app_state.gov.deposit_params.max_deposit_period="30s"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
jq '.app_state.gov.voting_params.voting_period="30s"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

# Set gas limit in genesis
jq '.consensus_params.block.max_gas="10000000"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

# Set claims start time
node_address=$($DNOM keys list | grep  "address: " | cut -c12-)
current_date=$(date -u +"%Y-%m-%dT%TZ")
jq -r --arg current_date "$current_date" '.app_state.claims.params.airdrop_start_time=$current_date' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

# Set claims records for validator account
amount_to_claim=10000
jq -r --arg node_address "$node_address" --arg amount_to_claim "$amount_to_claim" '.app_state.claims.claims_records=[{"initial_claimable_amount":$amount_to_claim, "actions_completed":[false, false, false, false],"address":$node_address}]' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

# Set claims decay
jq '.app_state.claims.params.duration_of_decay="1000000s"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
jq '.app_state.claims.params.duration_until_decay="100000s"' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

# Claim module account:
# 0xA61808Fe40fEb8B3433778BBC2ecECCAA47c8c47 || guru15cvq3ljql6utxseh0zau9m8ve2j8erz8lq9ma5
jq -r --arg amount_to_claim "$amount_to_claim" '.app_state.bank.balances += [{"address":"guru15cvq3ljql6utxseh0zau9m8ve2j8erz8lq9ma5","coins":[{"denom":"aguru", "amount":$amount_to_claim}]}]' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

# Set the distribution module moderator and base address:
jq -r --arg moderator_address "$($DNOM keys show $MODKEY --address --keyring-backend "$KEYRING" --home "$CHAINDIR")" '.app_state["distribution"]["moderator_address"] = $moderator_address' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
jq -r --arg base_address "$($DNOM keys show $BASEKEY --address --keyring-backend "$KEYRING" --home "$CHAINDIR")" '.app_state["distribution"]["base_address"] = $base_address' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
# jq -r --arg moderator_address "$($DNOM keys show $MODKEY --address --keyring-backend "$KEYRING" --home "$CHAINDIR")" '.app_state["distribution"]["moderator_address"]="guru17nywmzjr248agulkw584yt7qfvatq8dmp0ly06"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
# jq -r --arg base_address "$($DNOM keys show $MODKEY --address --keyring-backend "$KEYRING" --home "$CHAINDIR")" '.app_state["distribution"]["base_address"]="guru17nywmzjr248agulkw584yt7qfvatq8dmp0ly06"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"


# disable produce empty block
sed -i 's/create_empty_blocks = true/create_empty_blocks = false/g' "$CONFIG_TOML"

# Allocate genesis accounts (cosmos formatted addresses)
$DNOM add-genesis-account $KEY 100000000000000000000000000aguru --keyring-backend $KEYRING
$DNOM add-genesis-account $MODKEY 100000000000000000000000000aguru --keyring-backend $KEYRING
$DNOM add-genesis-account $BASEKEY 100000000000000000000000000aguru --keyring-backend $KEYRING
$DNOM add-genesis-account "tester1" 100000000000000000000000000aguru --keyring-backend $KEYRING
$DNOM add-genesis-account "tester2" 100000000000000000000000000aguru --keyring-backend $KEYRING
$DNOM add-genesis-account "tester3" 100000000000000000000000000aguru --keyring-backend $KEYRING

# define the balances for other accounts
for ((i=$START_INDEX_VAL+1; i<${#ADDRESSES[@]}; i++)); do
  ADDRESS="${ADDRESSES[i]}"
  jq -r --arg genesis_address "$ADDRESS" '.app_state.bank.balances += [{"address":$genesis_address,"coins":[{"denom":"aguru", "amount":"100000000000000000000000000"}]}]' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
done


# Update total supply with claim values
# Bc is required to add this big numbers
# total_supply=$(bc <<< "$amount_to_claim+$validators_supply")
total_supply=2100000000000000000000010000
jq -r --arg total_supply "$total_supply" '.app_state.bank.supply[0].amount=$total_supply' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"


# update genesis file as for given params
for element in "${GENESIS_PARAMS[@]}"; do
  variable=$(echo "$element" | sed -E 's/(.*)=.*/\1/')
  value=$(echo "$element" | sed -E 's/.*=(.*)/\1/')

  if [[ "$value" == "true" ]]; then
    jq --arg key "$variable" --argjson value true 'reduce ($key | split(".")) as $path (.; setpath($path; $value))' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  elif [[ "$value" == "false" ]]; then
    jq --arg key "$variable" --argjson value false 'reduce ($key | split(".")) as $path (.; setpath($path; $value))' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  else
    jq --arg key "$variable" --arg value "$value" 'reduce ($key | split(".")) as $path (.; setpath($path; $value))' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
  fi
done


# set custom pruning settings
if [ "$PRUNING" = "custom" ]; then
  sed -i 's/pruning = "default"/pruning = "custom"/g' "$APP_TOML"
  sed -i 's/pruning-keep-recent = "0"/pruning-keep-recent = "2"/g' "$APP_TOML"
  sed -i 's/pruning-interval = "0"/pruning-interval = "10"/g' "$APP_TOML"
fi

# make sure the localhost IP is 0.0.0.0
sed -i 's/pprof_laddr = "localhost:6060"/pprof_laddr = "0.0.0.0:6060"/g' "$CONFIG_TOML"
sed -i 's/127.0.0.1/0.0.0.0/g' "$APP_TOML"

# Sign genesis transaction
$DNOM gentx $KEY 1000000000000000000000aguru --keyring-backend $KEYRING --chain-id "$CHAINID"
## In case you want to create multiple validators at genesis
## 1. Back to `$DNOM keys add` step, init more keys
## 2. Back to `$DNOM add-genesis-account` step, add balance for those
## 3. Clone this ~/.gurud home directory into some others, let's say `~/.clonedGurud`
## 4. Run `gentx` in each of those folders
## 5. Copy the `gentx-*` folders under `~/.clonedGurud/config/gentx/` folders into the original `~/.gurud/config/gentx`

# Collect genesis tx
$DNOM collect-gentxs

# Run this to ensure everything worked and that the genesis file is setup correctly
$DNOM validate-genesis

# Start the node (remove the --pruning=nothing flag if historical queries are not needed)
$DNOM start "$TRACE" --log_level $LOGLEVEL --minimum-gas-prices=0.0001aguru --json-rpc.api eth,txpool,personal,net,debug,web3