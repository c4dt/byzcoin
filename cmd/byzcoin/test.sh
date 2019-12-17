#!/usr/bin/env bash

# Usage:
#   ./test [options]
# Options:
#   -b   re-builds bcadmin package

DBG_TEST=2
DBG_SRV=2
DBG_APP=2

NBR_SERVERS=4
NBR_SERVERS_GROUP=3

# Clears some env. variables
export -n BC_CONFIG
export -n BC
export BC_WAIT=true

root=../../upstream/cothority
. "./libtest.sh"

main(){
  startTest
  if [[ "$CLEANBUILD" = "yes" ]]; then
    build ../../../upstream/cothority/status
  fi
  run testVars
}

testVars(){
  setupVars
  testOK runBCConfig
  for f in private public; do
    testFile "$DATA_DIR/$f.toml"
    testGrep "$ADDRESS_NODE" cat "$DATA_DIR/$f.toml"
    testGrep "$DESCRIPTION" cat "$DATA_DIR/$f.toml"
  done
  testGrep "$WS_SSL_CHAIN" cat "$DATA_DIR/private.toml"
  testGrep "$WS_SSL_KEY" cat "$DATA_DIR/private.toml"
  testGrep "$BYZCOIN_ID" runBC show "$DATA_DIR"

  cp "$DATA_DIR/private.toml" private.toml

  ADDRESS_NODE=tls://byzcoin2.c4dt.org:7770
  testOK runBCConfig
  testGrep "$ADDRESS_NODE" cat "$DATA_DIR/private.toml"

  ADDRESS_WS=https://byzcoin3.c4dt.org:7770
  testOK runBCConfig
  testGrep "$ADDRESS_WS" cat "$DATA_DIR/private.toml"

  DESCRIPTION="Byzcoin node 2"
  testOK runBCConfig
  testGrep "$DESCRIPTION" cat "$DATA_DIR/private.toml"

  WS_SSL_KEY="$DATA_DIR/privkey2.pem"
  cp "${WS_SSL_KEY/2/}" "$WS_SSL_KEY"
  testOK runBCConfig
  testGrep "$WS_SSL_KEY" cat "$DATA_DIR/private.toml"

  WS_SSL_CHAIN="$DATA_DIR/fullchain2.pem"
  cp "${WS_SSL_CHAIN/2/}" "$WS_SSL_CHAIN"
  testOK runBCConfig
  testGrep "$WS_SSL_CHAIN" cat "$DATA_DIR/private.toml"

  BYZCOIN_ID="c8db5f60797d3bd44abb65d50e2deae5ac744e5ed49db6c9a05751b21179c22d"
  testOK runBCConfig
  testGrep "$BYZCOIN_ID" runBC show "$DATA_DIR"

  testOK cmp <( grep Private private.toml | sort ) \
    <( grep Private "$DATA_DIR/private.toml" | sort )
}

setupVars(){
  rm -rf "$CONODE_SERVICE_PATH"
  mkdir "$CONODE_SERVICE_PATH"
  export ADDRESS_NODE=tls://localhost:7770
  export ADDRESS_WS=https://localhost:7771
  export DESCRIPTION="New ByzCoin node"
  export DATA_DIR=./bc-data
  export WS_SSL_CHAIN="$DATA_DIR/fullchain.pem"
  export WS_SSL_KEY="$DATA_DIR/privkey.pem"
  export BYZCOIN_ID=9cc36071ccb902a1de7e0d21a2c176d73894b1cf88ae4cc2ba4c95cd76f474f3
  export DEBUG_LVL=2
  export DEBUG_COLOR=false
  export UPDATE_ONLY=true
  rm -rf "$DATA_DIR"
  mkdir "$DATA_DIR"
  # DO NOT USE THIS IN PRODUCTION - 512 bits is only for testing!
  openssl req -x509 -newkey rsa:512 -nodes -subj "/CN=localhost" \
    -keyout "$WS_SSL_KEY" -out "$WS_SSL_CHAIN" -days 1
}

runBC(){
  ./byzcoin "$@"
}

runBCConfig(){
  runBC config --address-node "$ADDRESS_NODE" \
    --address-ws "$ADDRESS_WS" --desc "$DESCRIPTION" \
    --ws-ssl-chain "$WS_SSL_CHAIN" --ws-ssl-key "$WS_SSL_KEY" \
    --byzcoin-id "$BYZCOIN_ID" --data-dir "$DATA_DIR"
}

main
