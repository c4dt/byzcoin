#!/bin/bash

INTERVAL=c4dt/byzcoin:v3.4.5-200722-1540
ROLLUP=c4dt/byzcoin:v3.4.5-200727-1200
NODES=$(seq -f "node%g" 4)
DATA=nodes
BLOCK_INTERVAL=5s

main(){
  cleanup
  docker_config_run
  mint_coins 1
  docker_replace
  switch_leader
  mint_coins 5
}

cleanup() {
  for n in $NODES; do
    docker rm -f $n
  done

  mkdir -p $DATA
  if [ ! -f $DATA/bcadmin ]; then
    echo "Creating bcadmin binary"
    (cd pkg/cothority && go build ./byzcoin/bcadmin && mv bcadmin ../../$DATA)
  fi

  cd $DATA || exit 1
  rm -rf node*
}

docker_start() {
  local image=$1
  local name=$2
  local port=$3
  local ports="$port-$((port + 1))"
  docker run -v $(pwd)/$name:/byzcoin -p $ports:$ports -d \
    --name $name -h $name -e DEBUG_TIME=true -e DEBUG_COLOR=true \
    --network nodes $image \
    ./byzcoin --debug 2 run /byzcoin
  docker logs -f $name >>logs 2>&1 &
}

docker_config_run(){
  docker network create nodes
  PORT=2000
  for n in $NODES; do
    echo "Configuring $n"
    docker run -ti -v $(pwd)/$n:/byzcoin $INTERVAL \
      ./byzcoin config --data-dir /byzcoin \
      --address-node "tls://$n:$((PORT))" \
      --address-ws "http://$n:$((PORT + 1))" --desc $n >/dev/null

    echo "Starting $n"
    docker_start $INTERVAL $n $PORT

    PORT=$((PORT + 10))
  done

  rm -f roster.toml
  for p in node*/public.toml; do
cat <<EOF >>roster.toml
[[servers]]
$(cat $p | sed 's/^/  /' | sed 's/Services/servers.Services/')

EOF
  done

  echo "Creating new chain"
  rm -f *.cfg
  ./bcadmin -c . create -i $BLOCK_INTERVAL roster.toml

  echo "Latest block"
  ./bcadmin debug list http://localhost:2001
}

docker_replace(){
  PORT=2000
  for n in $NODES; do
    echo "Replacing $n"
    docker rm -f $n
    docker_start $ROLLUP $n $PORT

    echo "Minting some coins"
    ./bcadmin mint bc-* key-* \
    559cd91debcb38952632b509ee5e00624deac7275c7a986ebbe35bc2a6e3dfad 100 &
    sleep 5

    PORT=$((PORT + 10))
  done

  ./bcadmin debug list -v http://localhost:2001
}

mint_coins(){
  for mint in $(seq $1); do
    echo "Minting some coins $mint"
    ./bcadmin mint bc-* key-* 559cd91debcb38952632b509ee5e00624deac7275c7a986ebbe35bc2a6e3dfad 100
  done
  echo "Latest block"
  ./bcadmin debug list http://localhost:2001
}

switch_leader(){
  PORT=2000
  for n in $NODES; do
    PORT_NEXT=$(( (((PORT-2000)+10) % 40) + 2001))
    echo "Switching leader $n - next port: $PORT_NEXT"
    docker rm -f $n

    LEADER=$n
    while [ $LEADER = $n ]; do
      echo "Minting some coins for leader $LEADER"
      ./bcadmin mint bc-* key-* \
        559cd91debcb38952632b509ee5e00624deac7275c7a986ebbe35bc2a6e3dfad 100

      LEADER=$( ./bcadmin debug list -v http://localhost:$PORT_NEXT |
        grep "Roster: " | sed -e "s;.*\[tls://\([^:]*\):.*;\1;" )
      echo "New leader is: $LEADER"
    done

    docker_start $ROLLUP $n $PORT
    sleep 2

    PORT=$((PORT + 10))
  done
}

main
#cd $DATA
#docker rm -f node4
#docker_start $ROLLUP node4 2030
#switch_leader
#docker_start $ROLLUP node4 2030
#docker rm -f node2
#
