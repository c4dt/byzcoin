# Byzcoin node

Byzcoin is a high-performance permissioned blockchain that can safely run in 
public mode, meaning everybody with access tokens can use it.
It is developed by the [DEDIS](https://dedis.epfl.ch) lab at 
[EPFL](https://epfl.ch) and supported by the [C4DT](https://c4dt.org).
Novel parts of the byzcoin blockchain are the 
[DARC](https://github.com/dedis/cothority/tree/master/darc) structures that 
allow delegation of access control and decentralized management.
Another new part is 
[Calypso](https://github.com/dedis/cothority/tree/master/calypso)
which allows to encrypt to a set of servers and define access control to 
users who will be able to decrypt the data. 

In this README you will find a short explanation how you can get a node up 
and running to join the DEDIS byzcoin [test-net](https://status.dedis.ch).
For a more technical description, see one of:
- [Why this repository](TECHNICAL.md)
- [Collective Authority Project](https://github.com/dedis/cothority)
- [ByzCoin Paper](https://eprint.iacr.org/2017/406.pdf)
- [Security Assessment](ASSESSMENT.md)

## Running your node

To run a byzcoin node, you need a server with 2GB of RAM, 10GB of harddisk, 
and a CPU not older than 5 years. 
The simplest way to run your node is using docker.
If you want to compile from source, or use the binary, please have a look at 
[more setups](SOURCE.md).

### TLDR

```bash
wget https://raw.githubusercontent.com/c4dt/byzcoin/master/docker-compose.yml
# Edit ADDRESS_* and DESCRIPTION
vi docker-compose.yaml
docker-compose up
```

And send `~/byzcoin/public.toml` to 
[byzcoin@groupes.epfl.ch](mailto:byzcoin@groupes.epfl.ch)

## Step-by-step instructions

To setup a new node, you'll need to do the following steps:

1. Configure your node
2. Starting your node
3. Sign up for the DEDIS network
4. Updating your node

## Configuring your node

The simplest way of making your local configuration is to use the 
docker-compose file found in the root-directory of the byzcoin repository. 
In it you find the following variables:

```bash
# ADDRESS_NODE should always be tls:// - tcp:// is insecure and should
# not be used.
- ADDRESS_NODE=tls://byzcoin.c4dt.org:7770
# ADDRESS_WS can be either http:// or https:// - for most of the use-cases
# you want this to be https://, so that secure webpages can access the node.
- ADDRESS_WS=https://byzcoin.c4dt.org:7771
# A short description of your node that will be visible to the outside.
- DESCRIPTION="New ByzCoin node"
# Only needed if ADDRESS_WS is https. Ignored if it is http. 
- WS_SSL_CHAIN=fullchain.pem
- WS_SSL_KEY=privkey.pem
# ID of the byzcoin to follow - this corresponds to the DEDIS byzcoin.
- BYZCOIN_ID=9cc36071ccb902a1de7e0d21a2c176d73894b1cf88ae4cc2ba4c95cd76f474f3
```

The following variables are OK as default and can be changed if needed:
```bash
# How much debugging output - 0 is none, 1 is important ones, 2 is 
# interesting, 3 is detailed, 4 is lots of details, and 5 is too detailed for
# most purposes.
- DEBUG_LVL=2
# Whether to niceify the debug outputs. If you put this to `true`, you should
# have a black background in the terminal.
- DEBUG_COLOR=false
# Send tracing information to honeycomb.io. The format is: api_key:dataset.
# If no key is set, tracing is disabled.
- HONEYCOMB_API_KEY=
```

Update it with the address of your node, and eventually copy the SSL-files
to the `~/byzcoin` directory. 
The example for the ssl-files is given for letsencrypt files.

If you're running your node for the DEDIS 9cc3, please get in contact with
linus.gasser@epfl.ch, so that he can send you his HONEYCOMB_API_KEY.
Then it'll be easier to trace eventual errors in the network.

## Starting your node

Starting the node for the first time is done like this:

```bash
docker-compose up
```

This starts the node and prints some debugging information.
If something goes wrong, an error is printed. 
Once the node is up and running, you can check it with:

```bash
go build go.dedis.ch/cothority/v3/status
status --host http://localhost:7771
```

Once it's running, start it in the background with:

```bash
docker-compose up -d
```

## Sign up for the DEDIS network

To get your node included in the DEDIS network, you need to sign the 
DEDIS_BYZCOIN.md file and send it, together with `~/byzcoin/public.toml` to 
[byzcoin@groupes.epfl.ch](mailto:byzcoin@groupes.epfl.ch).

## Updating your node

The `docker-compose.yaml` file contains a link to 
[watchtower](https://hub.docker.com/r/v2tec/watchtower/)
which will check every hour if there is a new docker-file available.

The current `docker-compose.yaml` contains a link to `byzcoin:latest` which 
is updated every day. 
To not have all the nodes update within one hour, it is better to link to one
 of the weekly snapshots: `byzcoin:Sun`, `byzcoin:Mon`, `byzcoin:Tue`, ..., 
 `byzcoin:Sat`.
By chosing a random day, the rebooting will be randomized over the week.  

## Running your node from crontab

Once your node is up and running, you can get it started automatically from 
your crontab. Add the following line:

```bash
@reboot docker-compose restart
```

# Migrating from old installation

If you're running a node with its own `private.toml` and `xxxxxx.db`, the easiest
way is to copy those two files in a `~/byzcoin` directory and then adjust the
following variables in `docker-compose.yaml`:

- ADDRESS_NODE=tls://byzcoin.c4dt.org:7770 # Change with your main address:port of the node
- ADDRESS_WS=http://byzcoin.c4dt.org       # This is how your node will listen to websocket requests
- USE_TLS=false                            # Set this to 'true' if the node should handle TLS connections

## ADDRESS_WS

This is taken from the point of view of the node. We suppose you have TLS _some_ where:

1. The node is behind a proxy that handles TLS (apache, nginx, traefik, sstunnel, ...).
`url:port` is where the proxy forwards the requests to the node. If you leave the `ADDRESS_WS`
empty, it will take the default value, which is the same as `ADDRESS_NODE`, but using the
next port.
```
  ADDRESS_WS=[https://url:port]
```
2. The node handles the certificates on its own. If you omit `port`, it defaults to `443`:
```
  ADDRESS_WS=https://url[:port]
  USE_TLS=true
```

## public.toml

The public.toml file is re-generated according to the `private.toml` file. This brings some problems
to the `ADDRESS_WS`, which is used for the `URL` parameter in the `public.toml`. If you have a proxy
in front of the node, and the `ADDRESS_WS` is different from the proxy address, the `URL` parameter
in `public.toml` will be wrong and you will have to adjust it manually.

## LetsEncrypt

In case the node handles its own certificates, and that you're using LetsEncrypt, you can
create the following script:
```
cat <<EOF > /etc/letsencrypt/renewal-hooks/deploy/byzcoin.sh
#!/bin/bash
cp /etc/letsencrypt/live/url/{fullchain,privkey}.pem /home/conode/byzcoin
EOF
chmod a+x /etc/letsencrypt/renewal-hooks/deploy/byzcoin.sh
```

## Traefik

If you have a working [dockerified trafik instance](https://hub.docker.com/_/traefik) on your server,
including a docker-wide network, you can add the following to your byzcoin service definition in
`docker-compose.yaml`:
```
    environment:
      # ...
      - ADDRESS_WS=
      # ...
    labels:
     - "traefik.enable=true"
     - "traefik.http.routers.byzcoin.rule=Host(`byzcoin.c4dt.org`)"
     - "traefik.http.routers.byzcoin.entrypoints=websecure"
     - "traefik.http.routers.byzcoin.tls.certresolver=myresolver"
     - "traefik.http.services.byzcoin.loadbalancer.server.port=7771"
    networks:
     - traefik
     
networks:
  traefik:
    external:
      name: traefik_traefik # Adjust to match your traefik-installation
```

Before publishing your `public.toml`, you need to change the `URL` to:
```
URL = https://byzcoin.c4dt.org
```

# Updating

Two github actions are used to update the code once per week and serve it
 over the coming week.
The first action is started once a week to update the repo with the latest code
 from the cothority and onet to produce a docker file.
The second action runs every day to update the link of the 'daily' docker
 image to the image from the first action.

Every node has currently a `watchtower` docker running once an hour and will
 update accordingly.
As every node is linked to a different 'daily' docker image, the 7 nodes
 update throughout the week.
 
## Verification

The following verifications are done before a new image is generated:
* a small unit-test for the binary
* the new code correctly replays all known transactions on the chain
* a cothority with 4 nodes can correctly migrate from the previous to the
 current nodes
 
All those tests are done in the `update.yaml` github action.
 
## Manual updates
 
During heavy development, the automatic update is sometimes disabled.
To update manually, use the following:

```bash
make update
git commit -am "updated to latest byzcoin"
git push
```

## Overwrite all images

As this is still under development, sometimes it's nice to be able to update
 all nodes without having to wait for a week.
This can be done using

```bash
make docker-push-all
```

This will create a new docker image and tag it for every weekday.
As the current nodes update their image once an hour, you'll have to wait for
 an hour for every node to update.
To mark these updates as somewhat not ideal, a `force` is added to the version.
