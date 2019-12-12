module github.com/c4dt/byzcoin

go 1.13

require (
	github.com/urfave/cli v1.22.2
	go.dedis.ch/cothority/v3 v3.4.0
	go.dedis.ch/kyber/v3 v3.0.11
	go.dedis.ch/onet/v3 v3.0.31
)

replace go.dedis.ch/cothority/v3 => ./pkg/cothority

replace go.dedis.ch/onet/v3 => ./pkg/onet
