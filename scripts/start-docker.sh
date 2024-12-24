#!/bin/bash

KEY="dev0"
CHAINID="guru_3110-1"
MONIKER="mymoniker"
DATA_DIR=$(mktemp -d -t guru-datadir.XXXXX)

echo "create and add new keys"
./gurud keys add $KEY --home $DATA_DIR --no-backup --chain-id $CHAINID --algo "eth_secp256k1" --keyring-backend test
echo "init Guru with moniker=$MONIKER and chain-id=$CHAINID"
./gurud init $MONIKER --chain-id $CHAINID --home $DATA_DIR
echo "prepare genesis: Allocate genesis accounts"
./gurud add-genesis-account \
"$(./gurud keys show $KEY -a --home $DATA_DIR --keyring-backend test)" 1000000000000000000aguru,1000000000000000000stake \
--home $DATA_DIR --keyring-backend test
echo "prepare genesis: Sign genesis transaction"
./gurud gentx $KEY 1000000000000000000stake --keyring-backend test --home $DATA_DIR --keyring-backend test --chain-id $CHAINID
echo "prepare genesis: Collect genesis tx"
./gurud collect-gentxs --home $DATA_DIR
echo "prepare genesis: Run validate-genesis to ensure everything worked and that the genesis file is setup correctly"
./gurud validate-genesis --home $DATA_DIR

echo "starting guru node $i in background ..."
./gurud start --pruning=nothing --rpc.unsafe \
--keyring-backend test --home $DATA_DIR \
>$DATA_DIR/node.log 2>&1 & disown

echo "started guru node"
tail -f /dev/null