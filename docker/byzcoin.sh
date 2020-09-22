#!/usr/bin/env bash

echo "Using command ${BYZCOIN:=./byzcoin}"

export DEBUG_LVL DEBUG_COLOR DEBUG_TIME
ssl=""
if [[ ( $ADDRESS_WS =~ https.* ) && ( $USE_TLS != 'false' ) ]]; then
    echo "Using TLS"
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
$BYZCOIN config --address-node "$ADDRESS_NODE" \
    --address-ws "$ADDRESS_WS" --desc "$DESCRIPTION" \
    --byzcoin-id "$BYZCOIN_ID" --data-dir /byzcoin $ssl
echo "Starting ByzCoin"

if [[ -z "$GRAYLOG" ]]; then
  echo "Running without Graylog"
  $BYZCOIN --debug $DEBUG_LVL run /byzcoin
else
  echo "Forwarding to Graylog: ${GRAYLOG/:/ }"
  $BYZCOIN --debug $DEBUG_LVL run /byzcoin | tee /dev/stderr | \
    netcat -v ${GRAYLOG/:/ }
fi
