module github.com/c4dt/byzcoin

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/urfave/cli v1.22.3
	go.dedis.ch/cothority/v3 v3.4.0
	go.dedis.ch/kyber/v3 v3.0.13
	go.dedis.ch/onet/v3 v3.2.10
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace go.dedis.ch/cothority/v3 => ./pkg/cothority

replace go.dedis.ch/onet/v3 => ./pkg/onet
