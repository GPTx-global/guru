#!/bin/bash

if [ $# -lt 4 ] ; then
	echo $'\nPlease input the persistent peers, mode, node index, and branch anme.'
	echo $'\n./init-node.sh [PEERS] [MODE] [NODE_INDEX] [BRANCH]\n'
	exit
fi

PEERS=$1
MODE=$2
NODE_INDEX=$3
BRANCH=$4

source env.sh

LOCAL_GENESIS="./genesis_$BRANCH.json" 
TARGET_GENESIS="$CHAINDIR/config/genesis.json"

KEY="dev0"
MNIC=""
if [ "$MODE" = "V" ]; then
    MNIC=${MNICS[$START_INDEX_VAL+$NODE_INDEX]}
else
    MNIC=${MNICS[$START_INDEX_FULL+$NODE_INDEX]}
fi


# validate dependencies are installed
command -v jq > /dev/null 2>&1 || { echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"; exit 1; }

# used to exit on first error (any non-zero exit code)
set -e

rm -rf "$CHAINDIR"

# Set client config
$DNOM config keyring-backend "$KEYRING"
$DNOM config chain-id "$CHAINID"

# if $KEY exists it should be deleted
echo "$MNIC" | $DNOM keys add "$KEY" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$CHAINDIR"
echo "${MNICS[1]}" | $DNOM keys add "$MODKEY" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$CHAINDIR"

# Set moniker and chain-id for Guru (Moniker can be anything, chain-id must be an integer)
$DNOM init "$MONIKER-node-$NODE_INDEX" --chain-id "$CHAINID"


# replace the genesis file
if [ ! -f "$LOCAL_GENESIS" ]; then
    echo "Error: genesis file not found at $LOCAL_GENESIS"
    exit 1
fi

# Backup existing genesis file
cp "$TARGET_GENESIS" "$TARGET_GENESIS.bak"

# Replace the genesis file
cp "$LOCAL_GENESIS" "$TARGET_GENESIS"


# disable produce empty block
sed -i 's/create_empty_blocks = true/create_empty_blocks = false/g' "$CONFIG_TOML"

# set custom pruning settings
if [ "$PRUNING" = "custom" ]; then
  sed -i 's/pruning = "default"/pruning = "custom"/g' "$APP_TOML"
  sed -i 's/pruning-keep-recent = "0"/pruning-keep-recent = "2"/g' "$APP_TOML"
  sed -i 's/pruning-interval = "0"/pruning-interval = "10"/g' "$APP_TOML"
fi

# add persistent peers
sed -i "s/persistent_peers = \"\"/persistent_peers = \"$PEERS\"/g" "$CONFIG_TOML"

# make sure the localhost IP is 0.0.0.0
sed -i 's/pprof_laddr = "localhost:6060"/pprof_laddr = "0.0.0.0:6060"/g' "$CONFIG_TOML"
sed -i 's/127.0.0.1/0.0.0.0/g' "$APP_TOML"

# Run this to ensure everything worked and that the genesis file is setup correctly
$DNOM validate-genesis

# Start the node (remove the --pruning=nothing flag if historical queries are not needed)
$DNOM start "$TRACE" --log_level $LOGLEVEL --minimum-gas-prices=0.0001aguru --json-rpc.api eth,txpool,personal,net,debug,web3