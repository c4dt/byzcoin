FROM golang:1.15 as builder-byzcoin

RUN apt update && apt install -y zsh && apt clean
COPY go.* /byzcoin/
COPY cmd /byzcoin/cmd/
COPY pkg /byzcoin/pkg/
WORKDIR /byzcoin
ENV LDFLAGS="-s -w -X main.gitTag=archive"
RUN go build -ldflags="$LDFLAGS" ./cmd/byzcoin
RUN go build -ldflags="$LDFLAGS" ./cmd/full
WORKDIR /byzcoin/pkg/cothority
RUN go build -ldflags="$LDFLAGS" ./byzcoin/bcadmin
RUN go build -ldflags="$LDFLAGS" ./personhood/phapp
RUN go build -ldflags="$LDFLAGS" ./calypso/csadmin
RUN cd scmgr && go build -ldflags="$LDFLAGS" .
WORKDIR /byzcoin
RUN cp  /byzcoin/pkg/cothority/bcadmin /byzcoin/pkg/cothority/phapp \
        /byzcoin/pkg/cothority/scmgr/scmgr \
        /byzcoin/pkg/cothority/csadmin /byzcoin
COPY docker/byzcoin.sh archive/setup_demo_chain.sh ./
RUN ./setup_demo_chain.sh

FROM node:12 as builder-omniledger
RUN git clone https://github.com/c4dt/omniledger -b archive /omniledger
WORKDIR /omniledger/webapp
RUN npm ci && npm link ../dynacred
RUN npx ng build --prod --base-href /login/ --deploy-url /login/ --aot --output-path www
COPY --from=builder-byzcoin /byzcoin/nodes/config.toml www/assets/

FROM node:14 as builder-olexplorer
RUN git clone https://github.com/c4dt/ol-explorer -b archive /ol-explorer
WORKDIR /ol-explorer
RUN npm ci
RUN npx ng build --prod --base-href /ol-explorer/ --deploy-url /ol-explorer/ --output-path www
COPY --from=builder-byzcoin /byzcoin/nodes/config.toml www/assets/config.toml

FROM node:14 as builder-columbus
RUN git clone https://github.com/c4dt/columbus-united -b archive /columbus
WORKDIR /columbus
RUN npm i
RUN npm run bundle
COPY --from=builder-byzcoin /byzcoin/nodes/config.toml assets/config.toml

FROM scratch as setup_log
COPY --from=builder-byzcoin /byzcoin/signup.link .
COPY --from=builder-byzcoin /byzcoin/nodes/config.toml .

FROM lipanski/docker-static-website:latest as web
#FROM python:latest
COPY archive/* ./
COPY --from=builder-omniledger /omniledger/webapp/www/ login/
COPY --from=builder-olexplorer /ol-explorer/www/ ol-explorer/
COPY --from=builder-columbus /columbus/ columbus/
COPY --from=builder-byzcoin /byzcoin/signup.link signup.link

FROM debian:bookworm-slim as byzcoin

RUN apt update && apt install -y procps ca-certificates netcat-openbsd && apt clean
WORKDIR /root/
RUN mkdir /byzcoin
RUN mkdir -p .local/share .config
RUN ln -s /byzcoin .local/share/conode
RUN ln -s /byzcoin .config/conode
COPY --from=builder-byzcoin /byzcoin/byzcoin /byzcoin/full \
                    /byzcoin/bcadmin /byzcoin/phapp \
                    /byzcoin/scmgr /byzcoin/csadmin /root/
COPY --from=builder-byzcoin /byzcoin/nodes/ /root/nodes/
COPY docker/byzcoin.sh /root/

ENV BYZCOIN=./full
ENV DEBUG_LVL=2
ENV DEBUG_COLOR=false
ENV DEBUG_TIME=true

CMD ./byzcoin.sh
