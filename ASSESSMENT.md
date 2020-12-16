# Security Assessment

This document gives a short overview of the security aspects when running a
 byzcoin node.
It is written by one of the main authors of byzcoin, so it should be taken
 with a big grain of salt.
Every contribution is very welcome.

I discuss the following security aspects of running a byzcoin node:
- remote execution
- maliciously inserted code
- DOS attacks against or using the server
- CPU / harddisk

Each node opens two ports:
- one port for node-to-node communication using protobuf-over-tls
- one port for web clients to connect, protected by a valid TLS certificate

TLDR: Look at https://github.com/c4dt/byzcoin/issues/14 for proposed fixes.

## Versions

- 20-02-12 - initial version with list of possible disk-filling attacks
- 20-12-16 - updated version with implemented fixes

## Remote Execution

Even though the byzcoin node is run in a docker container, the risk of
 executing non-desired code exists.
Usual attacks exploit a buffer overflow or badly tested input variables to
 make the server execute code from the attacker.
For this attack to work, the server software needs to do at least one of the
 following:
 
1. accept invalid messages over the network that lead to undesired code-paths
2. have a code-path that executes external binaries

For 1., the following points need to be taken into account
- using golang makes buffer overflows nearly impossible, as long as the code
 doesn't use `unsafe`
- network packets need to be safely parsed

Regarding 2., parts of the code that allow binary execution must be specially
 secured.
 
### Use of 'unsafe' in the code

To check whether 'unsafe' is used in the code, two parts need to be verified:
1. The code of the byzcoin node iteslf, in the `pkg/`-directory
2. All code in the libraries

To get all the code of all the libraries, it is a big advantage of having a
 golang project, as all libraries are available as source code.
When running `go build -v -a`, a list of all packages that are included in
 the final binary is printed.

Afterwards it's a simple execution of `grep` to check where `unsafe` is used:
- `cmd` - the actual command-line tool - no matches
- `pkg` - libraries from DEDIS - no matches

#### Use of 'unsafe' in dependencies

Here the main directories that match have `unsafe` operators in their code
, sorted from most concerning to least concerning.
- `golang.org/x/sys` - golang's system libraries - supposedly they know what
 they're doing
- `github.com/gorilla/websocket@v1.4.0` - websocket library - the code-path is
   actively used
- `go.etcd.io/bbolt@v1.3.3` - our database that is used internally
- `github.com/ethereum/go-ethereum@v1.8.27` - lots of code that is not in the
 active code-path
- `github.com/aristanetworks/goarista@v0.0.0-20191023202215-f096da5361bb
` - library for different things - used extensively in go-ethereum
- `github.com/syndtr/goleveldb@v1.0.0` - used by go-ethereum, but not in the
 active code-path
- `github.com/daviddengcn/go-colortext@v0.0.0-20180409174941-186a3d44e920
` - output colors to terminal - only `unsafe` in windows compilation
- `github.com/allegro/bigcache@v1.2.1` - conversion from byte to string
 - should be safe
- `github.com/prataprc/goparsec@v0.0.0-20180806094145-2600a2a4a410
` - bytes2string, seems to be safe

If we suppose that the golang-team knows what they're doing when using
 `unsafe`, this leaves us with the `bbolt`, the `websocket`, and the `go
 -ethereum` library and its dependencies.

For the `bbolt` library, the extensive use of `unsafe` is due to the high
 optimizations used in the library.
A full review of this code should be done to be really sure that none of the
 `unsafe` options in there can be used.

For the `websocket` library, the codepath is quite visible and should also be
 checked. A first visual inspection shows that there is a potential harm, as
  unsafe memory is modified.

The `go-ethereum` library has some dependencies that are not in the active
 codepath, other are. It would have to be well checked that all `unsafe` use
  is correct.

### Network packets parsing

All parsing of the incoming network packages is done using the `dedis
/protobuf` library. 
This assures that all incoming data is semantically correct and correspond to
 the actual structure requested by the code. 

### Command-execution code paths

In the main code there is no call to any go library with the goal of
 executing a command.
The only reference to `exec.Command` from the golang library is in `ufave/cli
/build.go`, but this file is marked inactive using `+build ignore`.

## Maliciously inserted code

It is always possible to have somebody insert code that will trigger a remote
 execution possibility for an attacker.
Contrary to `node`, where thousands of packages are used to build a single
 application, golang uses much less dependencies.
The total list of external (non-system) libraries used can be found in
 Appendix C.
The code of the byzcoin node itself is only updated after a code-review from
 the C4DT.

## DOS attacks

The byzcoin service itself is only badly protected against DOS attacks.
It is probably quite easy to send requests over WebSockets to spike the CPU
 to 100%.
However, clients without a valid byzcoin wallet cannot fill up the harddisk.

### Amplification attacks

There is the possibility of amplification attacks in the current version of
 byzcoin because any node can send any request for an existing protocol.
However, this touches only the nodes that are signed up for the current chain,
not nodes of other chains of arbitrary IP addresses.
For more details, see _Appendix B - Refuse non-byzcoin nodes_. 
 
## CPU / Harddisk

As the node is open to all clients, and they can ask any request to the node
, it is very easy to have the CPU going at 100%.
Currently we have no plan to remove that problem.
Should it come to pass that these attacks multiply, something like in
 _Appendix C - WebSocket shutdown_ would need to be implemented.
 
### Services capabilities

Each service in a byzcoin is open to unvalidated clients through a WebSocket
 interface.
As there is no rate limiting on calling the service endpoints, all of them can
be used to create a high CPU load on the server.
The issue https://github.com/dedis/cothority/issues/2207 describes a list of service
endpoints that should be amended:

- byzcoin
  - AddTransaction - can fill disk and blockchain with invalid transaction
- skipchain
  - StoreSkipBlock - can fill disk with the creation of new blockchains

### Protocol capabilities

In addition to services, byzcoin has a number of protocols that need to be
 run in order to work correctly.
Since the implementation of https://github.com/dedis/cothority/issues/2204,
 no amplification attacks to random IP addresses are possible anymore.

- byzcoinx
- skipchain
  - ProtocolExtendRoster
  - ProtocolGetBlocks
- blscosi
  - blscosi - deprecated - https://github.com/c4dt/byzcoin/issues/13
  - bdnprotocol
- messaging
  - broadcast
  - propagation

# Appendixes
 
## Appendix A - Suggested code reviews

While the main code should be safe from attacks, the code in `go.etcd.io
/bbolt` and `github.com/gorilla/websocket` should be reviewed with regard to
 the use of the `unsafe` keyword.

## Appendix B - Suggested changes

### Refuse invalid ClientTransactions

Currently anybody can send invalid ClientTransactions and thus fill up the
 blockchain.
The following proposal should be implemented:
https://github.com/dedis/cothority/issues/2205

### Limit some of byzcoin service endpoints

See https://github.com/dedis/cothority/issues/2207

## Appendix C - Possible changes

### WebSocket shutdown

If there is a DOS attack on the nodes where invalid clients flood the nodes
 with bogous requests, the following might be needed:
- remove the websocket access
- create follower nodes that only reply to requests, but not participate in
 the consensus
- add authentication to the follower nodes

## Appendix D - Complete list of external packages

```bash
go build -v -a 2> packages
sort packages | egrep "(gopkg.in|go.etcd.io|go.dedis|github.com)" | \
  sed -e "s/\([^\/]*.\)\([^\/]*.\)\([^\/]*\).*/\1\2\3/" | uniq \
  | tee packages.external
```
```text
github.com/BurntSushi/toml
github.com/allegro/bigcache
github.com/aristanetworks/goarista
github.com/c4dt/byzcoin
github.com/cpuguy83/go-md2man
github.com/daviddengcn/go-colortext
github.com/deckarep/golang-set
github.com/ethereum/go-ethereum
github.com/go-stack/stack
github.com/golang/snappy
github.com/gorilla/websocket
github.com/hashicorp/golang-lru
github.com/prataprc/goparsec
github.com/rs/cors
github.com/russross/blackfriday
github.com/shurcooL/sanitized_anchor_name
github.com/syndtr/goleveldb
github.com/urfave/cli
go.dedis.ch/cothority/v3
go.dedis.ch/fixbuf
go.dedis.ch/kyber/v3
go.dedis.ch/onet/v3
go.dedis.ch/protobuf
go.etcd.io/bbolt
gopkg.in/satori/go.uuid.v1
gopkg.in/tylerb/graceful.v1
```

## Appendix E - List of go-files compiled with `unsafe`

The list of files using unsafe has been created in the following way:

```bash
grep -r '"unsafe"' $( go list -m -f '{{.Dir}}' $( cat packages.external ) ) | grep .go: > unsafe.matches
grep -v Binary unsafe.matches | sed -e "s/.*go.mod.\(.*\):.*//" | uniq
```
```text
github.com/allegro/bigcache@v1.2.1/bytes.go:	"unsafe"
github.com/aristanetworks/goarista@v0.0.0-20191023202215-f096da5361bb/test/pretty_test.go:	"unsafe"
github.com/aristanetworks/goarista@v0.0.0-20191023202215-f096da5361bb/monotime/nanotime.go:	_ "unsafe" // required to use //go:linkname
github.com/aristanetworks/goarista@v0.0.0-20191023202215-f096da5361bb/key/composite.go:	"unsafe"
github.com/aristanetworks/goarista@v0.0.0-20191023202215-f096da5361bb/key/hash.go:import "unsafe"
github.com/aristanetworks/goarista@v0.0.0-20191023202215-f096da5361bb/areflect/force.go:	"unsafe"
github.com/aristanetworks/goarista@v0.0.0-20191023202215-f096da5361bb/sizeof/sizeof.go:	"unsafe"
github.com/aristanetworks/goarista@v0.0.0-20191023202215-f096da5361bb/sizeof/sizeof_test.go:	"unsafe"
github.com/daviddengcn/go-colortext@v0.0.0-20180409174941-186a3d44e920/ct_win.go:	"unsafe"
github.com/ethereum/go-ethereum@v1.8.27/consensus/ethash/algorithm.go:	"unsafe"
github.com/ethereum/go-ethereum@v1.8.27/consensus/ethash/ethash.go:	"unsafe"
github.com/ethereum/go-ethereum@v1.8.27/swarm/storage/feed/feed.go:	"unsafe"
github.com/ethereum/go-ethereum@v1.8.27/crypto/secp256k1/curve.go:	"unsafe"
github.com/ethereum/go-ethereum@v1.8.27/crypto/secp256k1/panic_cb.go:import "unsafe"
github.com/ethereum/go-ethereum@v1.8.27/crypto/secp256k1/secp256.go:	"unsafe"
github.com/ethereum/go-ethereum@v1.8.27/core/types/receipt.go:	"unsafe"
github.com/ethereum/go-ethereum@v1.8.27/core/types/block.go:	"unsafe"
github.com/ethereum/go-ethereum@v1.8.27/common/bitutil/bitutil.go:	"unsafe"
github.com/ethereum/go-ethereum@v1.8.27/p2p/discv5/sim_testmain_test.go:	"unsafe"
github.com/ethereum/go-ethereum@v1.8.27/log/handler_go13.go:	"unsafe"
github.com/ethereum/go-ethereum@v1.8.27/eth/tracers/tracer.go:	"unsafe"
github.com/gorilla/websocket@v1.4.0/mask.go:import "unsafe"
github.com/prataprc/goparsec@v0.0.0-20180806094145-2600a2a4a410/scanner.go:import "unsafe"
github.com/syndtr/goleveldb@v1.0.0/leveldb/version.go:	"unsafe"
github.com/syndtr/goleveldb@v1.0.0/leveldb/cache/cache_test.go:	"unsafe"
github.com/syndtr/goleveldb@v1.0.0/leveldb/cache/cache.go:	"unsafe"
github.com/syndtr/goleveldb@v1.0.0/leveldb/cache/lru.go:	"unsafe"
github.com/syndtr/goleveldb@v1.0.0/leveldb/db_test.go:	"unsafe"
github.com/syndtr/goleveldb@v1.0.0/leveldb/storage/file_storage_windows.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/cmd/bbolt/main.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/bucket.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/db.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/bolt_unix_solaris.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/db_test.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/bolt_unix.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/page.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/bolt_openbsd.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/bolt_arm.go:import "unsafe"
go.etcd.io/bbolt@v1.3.3/bolt_windows.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/freelist.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/node_test.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/tx.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/freelist_test.go:	"unsafe"
go.etcd.io/bbolt@v1.3.3/node.go:	"unsafe"
```
