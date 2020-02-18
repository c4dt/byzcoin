module github.com/c4dt/byzcoin

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/urfave/cli v1.22.2
	go.dedis.ch/cothority/v3 v3.4.0
	go.dedis.ch/kyber/v3 v3.0.12
	go.dedis.ch/onet/v3 v3.1.0
	golang.org/x/xerrors v0.0.0-20191011141410-1b5146add898
)

replace go.dedis.ch/cothority/v3 => ./pkg/cothority

replace go.dedis.ch/onet/v3 => ./pkg/onet
