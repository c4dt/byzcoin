#!/usr/bin/env bash

# Usage:
#   ./test [options]
# Options:
#   -b   re-builds bcadmin package

DBG_TEST=1
DBG_SRV=2
DBG_APP=2

NBR_SERVERS=4
NBR_SERVERS_GROUP=3

# Clears some env. variables
export -n BC_CONFIG
export -n BC
export BC_WAIT=true

. "../../upstream/cothority/libtest.sh"

main(){
  startTest
  if [[ "$CLEANBUILD" = "yes" ]]; then
    build ../../../upstream/cothority/status
  fi
  run testVars
}

testVars(){
  setupVars
  rm -rf $DATA_DIR
  runBC
  for f in private public; do
    testFile $DATA_DIR/$f.toml
    testGrep byzcoin.c4dt.org $DATA_DIR/$f.toml
    testGrep "New ByzCoin node" $DATA_DIR/$f.toml
  done
}

setupVars(){
  export ADDRESS_NODE=tls://byzcoin.c4dt.org:7770
  export ADDRESS_WS=https://byzcoin.c4dt.org:7771
  export DESCRIPTION="New ByzCoin node"
  export WS_SSL_CHAIN=fullchain.pem
  export WS_SSL_KEY=privkey.pem
  export BYZCOIN_ID=9cc36071ccb902a1de7e0d21a2c176d73894b1cf88ae4cc2ba4c95cd76f474f3
  export DATA_DIR=./bc-data
  export DEBUG_LVL=2
  export DEBUG_COLOR=false
  export UPDATE_ONLY=true
}

runBC(){
  ./byzcoin server "$@"
}

main
