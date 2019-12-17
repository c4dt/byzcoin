#!/usr/bin/env bash

export DEBUG_LVL DEBUG_COLOR
ssl=""
if [[ $ADDRESS_WS == https* ]]; then
    WS_CHAIN="/byzcoin/$WS_SSL_CHAIN"
    WS_KEY="/byzcoin/$WS_SSL_KEY"
    if [[ ! -f $WS_CHAIN ]]; then
        echo "Couldn't find SSL-certificate $WS_SSL_CHAIN, please provide it."
        exit 1
    fi
    if [[ ! -f $WS_KEY ]]; then
        echo "Couldn't find SSL-key $WS_SSL_KEY, please provide it."
        exit 1
    fi
    ssl="--ws-ssl-chain $WS_CHAIN \
        --ws-ssl-key $WS_KEY"
fi

echo "Configuring ByzCoin"
./byzcoin config --address-node "$ADDRESS_NODE" \
    --address-ws "$ADDRESS_WS" --desc "$DESCRIPTION" \
    --byzcoin-id "$BYZCOIN_ID" --data-dir /byzcoin $ssl
echo "Starting ByzCoin"
./byzcoin --debug $DEBUG_LVL run /byzcoin
