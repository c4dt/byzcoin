# Technical Details

## Why this directory

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

## What is included here?

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

### Protocols

TODO: list of protocols

## Ports

A byzcoin node needs two ports, and both should be exposed to the internet.
The standard port-numbers are 7770 and 7771, but other port numbers can be 
chosen.

- node-2-node communication: this uses a proprietary protocol where 
protobuf-messages are sent over a plain TCP or TLS connection
- node-2-client communication: also a proprietary protocol with protobuf-
messages over a websocket connection
