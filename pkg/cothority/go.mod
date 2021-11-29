module go.dedis.ch/cothority/v3

go 1.14

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/allegro/bigcache v1.2.1 // indirect
	github.com/c4dt/qrgo v0.0.0-20210312092726-8242850e1027
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/daviddengcn/go-colortext v1.0.0 // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/ethereum/go-ethereum v1.10.10
	github.com/prataprc/goparsec v0.0.0-20180806094145-2600a2a4a410
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli v1.22.3
	go.dedis.ch/kyber/v3 v3.0.13
	go.dedis.ch/onet/v3 v3.2.10
	go.dedis.ch/protobuf v1.0.11
	go.etcd.io/bbolt v1.3.4
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace go.dedis.ch/onet/v3 => ../onet
