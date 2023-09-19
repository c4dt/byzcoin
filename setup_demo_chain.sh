#!/bin/zsh

NODES_DIR=/byzcoin/nodes
mkdir -p $NODES_DIR/node-{1,2,3,4}
export BYZCOIN=./full \
 DEBUG_LVL=2 \
 DEBUG_COLOR=false \
 DEBUG_TIME=true
for node in $( seq 4 ); do
  PORT_N=$(( 7770 + node * 2 ))
  PORT_W=$(( PORT_N + 1 ))
  ADDRESS_NODE=tls://localhost:$PORT_N \
         ADDRESS_WS=http://localhost:$PORT_W \
         DESCRIPTION="Local Node $node" \
         DATA_DIR=$NODES_DIR/node-$node \
         ./byzcoin.sh &
done

# Waiting for the nodes to come up
# shellcheck disable=SC2046
# shellcheck disable=SC2005
# shellcheck disable=SC2012
while [[ "$(echo $(ls nodes/node*/*.db | wc -l))" != 4 ]]; do sleep 1; done
sleep 1
for node in nodes/node*; do
  echo -e "\n[[servers]]" >> nodes/group.toml
  sed -e "s/Services/servers.Services/" $node/public.toml >> nodes/group.toml
done

# Initializing a new byzcoin chain and creating a user
./bcadmin -c $NODES_DIR/ create $NODES_DIR/group.toml
sleep 1
BC=/$( ls $NODES_DIR/bc*.cfg )
KEY=/$( ls $NODES_DIR/key*.cfg )
URL=http://localhost:8080/login/register/device
./phapp user "$BC" "$KEY" $URL demo | tee login.tmp
tail -n 1 login.tmp | sed -e "s/.*is: //" > login.txt
rm login.tmp

# Create configuration files
echo -e "\nByzCoinID = \"${BC//(\/$NODES_DIR\/bc-|.cfg)/}\"" > nodes/config.toml
# TODO: add a real LTS
echo -e "LTSID = \"${BC//(\/$NODES_DIR\/bc-|.cfg)/}\"\n" >> nodes/config.toml
cat nodes/group.toml >> nodes/config.toml
pkill full
ls -R $NODES_DIR
