#!/bin/bash

KEYS[0]="dev0"
KEYS[1]="dev1"
KEYS[2]="dev2"
CHAINID="guru_3111-1"
MONIKER="localtestnet"
# Remember to change to other types of keyring like 'file' in-case exposing to outside world,
# otherwise your balance will be wiped quickly
# The keyring test does not require private key to steal tokens from you
KEYRING="test"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
# Set dedicated home directory for the gurud instance
HOMEDIR="$HOME/.tmp-gurud2-hermes"
# to trace evm
#TRACE="--trace"
TRACE=""

# Path variables
CONFIG=$HOMEDIR/config/config.toml
APP_TOML=$HOMEDIR/config/app.toml
GENESIS=$HOMEDIR/config/genesis.json
TMP_GENESIS=$HOMEDIR/config/tmp_genesis.json
DNOM="./build/gurud"

# validate dependencies are installed
command -v jq >/dev/null 2>&1 || {
	echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"
	exit 1
}

# used to exit on first error (any non-zero exit code)
set -e

# Reinstall daemon
make install

# User prompt if an existing local node configuration is found.
if [ -d "$HOMEDIR" ]; then
	printf "\nAn existing folder at '%s' was found. You can choose to delete this folder and start a new local node with new keys from genesis. When declined, the existing local node is started. \n" "$HOMEDIR"
	echo "Overwrite the existing configuration and start a new local node? [y/n]"
	read -r overwrite
else
	overwrite="Y"
fi


# Setup local node if overwrite is set to Yes, otherwise skip setup
if [[ $overwrite == "y" || $overwrite == "Y" ]]; then
	# Remove the previous folder
	rm -rf "$HOMEDIR"

	# Set client config
	$DNOM config keyring-backend $KEYRING --home "$HOMEDIR"
	$DNOM config chain-id $CHAINID --home "$HOMEDIR"

	# If keys exist they should be deleted
	# for KEY in "${KEYS[@]}"; do
	# 	$DNOM keys add "$KEY" --keyring-backend $KEYRING --algo $KEYALGO --home "$HOMEDIR"
	# done
	
	VAL_KEY=${KEYS[0]}
	VAL_MNEMONIC="legal lottery snack disease medal endorse enhance rotate rapid sibling never frequent canyon tell monkey amused stool bar dune mean potato farm panther bachelor"

	# dev1 address 0xc6fe5d33615a1c52c08018c47e8bc53646a0e101 | usdx1cml96vmptgw99syqrrz8az79xer2pcgpsm0z4g
	# pk 88cbead91aee890d27bf06e003ade3d4e952427e88f88d31d61d3ef5e5d54305
	DEV1_KEY=${KEYS[1]}
	DEV1_MNEMONIC="spot metal february journey isolate embark what life kite slim mean genuine pony anchor virtual employ virus project live inform knock poem balcony gown"

	# dev2 address 0x963ebdf2e1f8db8707d05fc75bfeffba1b5bac17 | usdx1jcltmuhplrdcwp7stlr4hlhlhgd4htqhxns2em
	# pk 741de4f8988ea941d3ff0287911ca4074e62b7d45c991a51186455366f10b544
	DEV2_KEY=${KEYS[2]}
	DEV2_MNEMONIC="useless flock hole topple dentist state clown mask devote since dignity shrug scout royal letter rare runway enable more lake they swing foam torch"
	
	# If keys exist they should be deleted
	echo "$VAL_MNEMONIC" | $DNOM keys add "$VAL_KEY" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"
	echo "$DEV1_MNEMONIC" | $DNOM keys add "$DEV1_KEY" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"
	echo "$DEV2_MNEMONIC" | $DNOM keys add "$DEV2_KEY" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"
	

	# Set moniker and chain-id for Guru (Moniker can be anything, chain-id must be an integer)
	$DNOM init $MONIKER -o --chain-id $CHAINID --home "$HOMEDIR"

	# Change parameter token denominations to aguru
	jq '.app_state["staking"]["params"]["bond_denom"]="aguru"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["crisis"]["constant_fee"]["denom"]="aguru"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="aguru"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["evm"]["params"]["evm_denom"]="aguru"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["inflation"]["params"]["mint_denom"]="aguru"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set gas limit in genesis
	jq '.consensus_params["block"]["max_gas"]="10000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set claims start time
	current_date=$(date -u +"%Y-%m-%dT%TZ")
	jq -r --arg current_date "$current_date" '.app_state["claims"]["params"]["airdrop_start_time"]=$current_date' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set claims records for validator account
	amount_to_claim=10000
	claims_key=${KEYS[0]}
	node_address=$($DNOM keys show "$claims_key" --keyring-backend $KEYRING --home "$HOMEDIR" | grep "address" | cut -c12-)
	jq -r --arg node_address "$node_address" --arg amount_to_claim "$amount_to_claim" '.app_state["claims"]["claims_records"]=[{"initial_claimable_amount":$amount_to_claim, "actions_completed":[false, false, false, false],"address":$node_address}]' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set claims decay
	jq '.app_state["claims"]["params"]["duration_of_decay"]="1000000s"' >"$TMP_GENESIS" "$GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["claims"]["params"]["duration_until_decay"]="100000s"' >"$TMP_GENESIS" "$GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Claim module account:
	# 0xA61808Fe40fEb8B3433778BBC2ecECCAA47c8c47 || guru15cvq3ljql6utxseh0zau9m8ve2j8erz8lq9ma5
	jq -r --arg amount_to_claim "$amount_to_claim" '.app_state["bank"]["balances"] += [{"address":"guru15cvq3ljql6utxseh0zau9m8ve2j8erz8lq9ma5","coins":[{"denom":"aguru", "amount":$amount_to_claim}]}]' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set the distribution module moderator and base address:
	mod_key=${KEYS[2]}
	base_key=${KEYS[2]}
	jq -r --arg moderator_address "$($DNOM keys show $mod_key --address --keyring-backend "$KEYRING" --home "$HOMEDIR")" '.app_state["distribution"]["moderator_address"] = $moderator_address' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq -r --arg base_address "$($DNOM keys show $base_key --address --keyring-backend "$KEYRING" --home "$HOMEDIR")" '.app_state["distribution"]["base_address"] = $base_address' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set the ica controller port
	jq '.app_state["interchainaccounts"]["controller_genesis_state"]["ports"] = ["icacontroller"]' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# set unbonding time to 2m
	# jq -r --arg unbonding_time "120s" '.app_state["staking"]["params"]["unbonding_time"] = $unbonding_time' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"


	if [[ $1 == "pending" ]]; then
		if [[ "$OSTYPE" == "darwin"* ]]; then
			sed -i '' 's/timeout_propose = "3s"/timeout_propose = "30s"/g' "$CONFIG"
			sed -i '' 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' "$CONFIG"
			sed -i '' 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' "$CONFIG"
			sed -i '' 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' "$CONFIG"
			sed -i '' 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' "$CONFIG"
			sed -i '' 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' "$CONFIG"
			sed -i '' 's/timeout_commit = "5s"/timeout_commit = "150s"/g' "$CONFIG"
			sed -i '' 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' "$CONFIG"
		else
			sed -i 's/timeout_propose = "3s"/timeout_propose = "30s"/g' "$CONFIG"
			sed -i 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' "$CONFIG"
			sed -i 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' "$CONFIG"
			sed -i 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' "$CONFIG"
			sed -i 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' "$CONFIG"
			sed -i 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' "$CONFIG"
			sed -i 's/timeout_commit = "5s"/timeout_commit = "150s"/g' "$CONFIG"
			sed -i 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' "$CONFIG"
		fi
	fi

    # enable prometheus metrics
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/prometheus = false/prometheus = true/' "$CONFIG"
        sed -i '' 's/prometheus-retention-time = 0/prometheus-retention-time  = 1000000000000/g' "$APP_TOML"
        sed -i '' 's/enabled = false/enabled = true/g' "$APP_TOML"
    else
        sed -i 's/prometheus = false/prometheus = true/' "$CONFIG"
        sed -i 's/prometheus-retention-time  = "0"/prometheus-retention-time  = "1000000000000"/g' "$APP_TOML"
        sed -i 's/enabled = false/enabled = true/g' "$APP_TOML"
    fi
	
	# Change proposal periods to pass within a reasonable time for local testing
	sed -i.bak 's/"max_deposit_period": "172800s"/"max_deposit_period": "30s"/g' "$HOMEDIR"/config/genesis.json
	sed -i.bak 's/"voting_period": "172800s"/"voting_period": "30s"/g' "$HOMEDIR"/config/genesis.json

	# set custom pruning settings
	sed -i.bak 's/pruning = "default"/pruning = "custom"/g' "$APP_TOML"
	sed -i.bak 's/pruning-keep-recent = "0"/pruning-keep-recent = "2"/g' "$APP_TOML"
	sed -i.bak 's/pruning-interval = "0"/pruning-interval = "10"/g' "$APP_TOML"

	# Allocate genesis accounts (cosmos formatted addresses)
	for KEY in "${KEYS[@]}"; do
		$DNOM add-genesis-account "$KEY" 100000000000000000000000000aguru --keyring-backend $KEYRING --home "$HOMEDIR"
	done

	# bc is required to add these big numbers
	total_supply=$(echo "${#KEYS[@]} * 100000000000000000000000000 + $amount_to_claim" | bc)
	jq -r --arg total_supply "$total_supply" '.app_state["bank"]["supply"][0]["amount"]=$total_supply' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Sign genesis transaction
	$DNOM gentx "${KEYS[0]}" 1000000000000000000000aguru --keyring-backend $KEYRING --chain-id $CHAINID --home "$HOMEDIR"
	## In case you want to create multiple validators at genesis
	## 1. Back to `gurud keys add` step, init more keys
	## 2. Back to `gurud add-genesis-account` step, add balance for those
	## 3. Clone this ~/.gurud home directory into some others, let's say `~/.clonedGurud`
	## 4. Run `gentx` in each of those folders
	## 5. Copy the `gentx-*` folders under `~/.clonedGurud/config/gentx/` folders into the original `~/.gurud/config/gentx`

	# Collect genesis tx
	$DNOM collect-gentxs --home "$HOMEDIR"

	# Run this to ensure everything worked and that the genesis file is setup correctly
	$DNOM validate-genesis --home "$HOMEDIR"

	if [[ $1 == "pending" ]]; then
		echo "pending mode is on, please wait for the first block committed."
	fi
fi

# Start the node (remove the --pruning=nothing flag if historical queries are not needed)
$DNOM start --metrics "$TRACE" --log_level $LOGLEVEL --minimum-gas-prices=0.0001aguru --json-rpc.api eth,txpool,personal,net,debug,web3 --api.enable --home "$HOMEDIR"
