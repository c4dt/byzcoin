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
# Send the logging information to the c4dt logger. Optional, can be put to
# "" if not needed.
- GRAYLOG=graylog.c4dt.org:9001
```

Update it with the address of your node, and eventually copy the SSL-files
to the `~/byzcoin` directory. 
The example for the ssl-files is given for letsencrypt files.

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

