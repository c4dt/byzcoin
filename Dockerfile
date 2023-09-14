FROM golang:1.15 as builder

COPY . /byzcoin
WORKDIR /byzcoin
ENV LDFLAGS="-s -w -X main.gitTag=archive"
RUN go build -ldflags="$LDFLAGS" ./cmd/byzcoin
RUN go build -ldflags="$LDFLAGS" ./cmd/full
WORKDIR /byzcoin/pkg/cothority
RUN go build -ldflags="$LDFLAGS" ./byzcoin/bcadmin
RUN go build -ldflags="$LDFLAGS" ./personhood/phapp
RUN cd scmgr && go build -ldflags="$LDFLAGS" .

FROM debian:bookworm-slim

RUN apt update && apt install -y procps ca-certificates netcat-openbsd && apt clean
WORKDIR /root/
RUN mkdir /byzcoin
RUN mkdir -p .local/share .config
RUN ln -s /byzcoin .local/share/conode
RUN ln -s /byzcoin .config/conode
COPY --from=builder /byzcoin/byzcoin /byzcoin/full /root/
COPY --from=builder /byzcoin/pkg/cothority/bcadmin /byzcoin/pkg/cothority/phapp /root/
COPY --from=builder /byzcoin/pkg/cothority/scmgr/scmgr /root/
COPY docker/byzcoin.sh /root/

CMD ["./byzcoin.sh"]
