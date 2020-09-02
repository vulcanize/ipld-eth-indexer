#!/bin/sh

set -ex

# clean up
trap 'killall geth && rm -rf "$TMPDIR"' EXIT
trap "exit 1" SIGINT SIGTERM

TMPDIR=$(mktemp -d)
echo "Temp dir is $TMPDIR"
dapp testnet --rpc-addr 0.0.0.0 --dir "$TMPDIR" > geth.log 2>&1 &
# give it a few secs to start up
sleep 90

read -r ACC BAL <<< "$(seth ls --keystore "$TMPDIR/8545/keystore")"
echo $ACC
echo $BAL


# Deploy a contract:
solc --bin --bin-runtime stateful.sol -o "$TMPDIR"
A_ADDR=$(seth send --create "$(<"$TMPDIR"/A.bin)" "constructor(uint y)" 1 --from "$ACC" --keystore "$TMPDIR"/8545/keystore --password /dev/null --gas 0xffffffff)

echo $A_ADDR

# Call transaction

TX=$(seth send "$A_ADDR" "off()" --gas 0xffff --password /dev/null --from "$ACC" --keystore "$TMPDIR"/8545/keystore --async)
echo $TX
RESULT=$(seth run-tx "$TX")
echo $RESULT