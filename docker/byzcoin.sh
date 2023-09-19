#!/usr/bin/env bash

export DEBUG_LVL DEBUG_COLOR DEBUG_TIME
BYZCOIN="${BYZCOIN:-./byzcoin}"
DATA_DIR="${DATA_DIR:-/byzcoin}"

config() {
  ssl=""
  if [[ ($ADDRESS_WS =~ https.*) && ($USE_TLS != 'false') ]]; then
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
    --byzcoin-id "$BYZCOIN_ID" --data-dir "$DATA_DIR" $ssl
}

if ! [[ -f /byzcoin/public.toml ]]; then
  if [[ "$NODE" ]]; then
    cp -a /root/nodes/node-"$NODE"/* /byzcoin
  else
    config
  fi
fi

echo "DATA_DIR is ${DATA_DIR}"

if [[ -z "$GRAYLOG" ]]; then
  echo "Running without Graylog"
  $BYZCOIN --debug $DEBUG_LVL run "$DATA_DIR"
else
  echo "Forwarding to Graylog: ${GRAYLOG/:/ }"
  $BYZCOIN --debug $DEBUG_LVL run "$DATA_DIR" | tee /dev/stderr |
    netcat -v ${GRAYLOG/:/ }
fi
