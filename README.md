# Byzcoin server

This directory is a copy of the code found in 
https://github.com/dedis/cothority 
with the goal to make it small enough to be code-reviewable.
The flow for code in this directory should always be:

```
dedis/cothority/byzcoin + patches ->
c4dt/byzcoin
```

The only binary in this directory is the server itself. For all other 
binaries used normally in byzcoin production, like `bcadmin`, `scmgr`, and 
others, please install them from the `dedis/cothority` directory.

## Running your node

To run a byzcoin node, you need a server with 2GB of RAM, 10GB of harddisk, 
and a CPU not older than 5 years. 
You can run the node either using docker, or start the binary directly.
Here is the setup to do in order to follow the DEDIS byzcoin instance.
For setting up your own instance, please have a look at
https://github.com/dedis/cothority/tree/master/conode

To setup a new node, you'll need to do the following steps:

1. Create a local configuration
2. Run and secure the node
3. Send the configuration to DEDIS for inclusion
4. Keep up-to-date with the latest version

### Create a local configuration

### Run and secure the node
### Send the configuration to DEDIS for inclusion
### Keep up-to-date with the latest version

## Technical Details

For those familiar with the cothority/onet project, here some technical 
details of what is in this repository.
This information is also useful as a starting point for a code review.

### Services

The cothority project uses onet for services available to the outside.
Byzcoin needs at least the following services:

- skipchain - to handle the underlying consensus protocol
- byzcoin - the actual transaction and global state part

For convenience, the following service is also added:

- status - return information about the node

### Ports

A byzcoin node needs two ports, and both should be exposed to the internet.
The standard port-numbers are 7770 and 7771, but other port numbers can be 
chosen.

- node-2-node communication: this uses a proprietary protocol where 
protobuf-messages are sent over a plain TCP or TLS connection
- node-2-client communication: also a proprietary protocol with protobuf-
messages over a websocket connection
