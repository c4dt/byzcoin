FROM debian:stretch-slim
RUN apt update && apt install -y procps ca-certificates netcat && apt clean
WORKDIR /root/
RUN mkdir /byzcoin
RUN mkdir -p .local/share .config
RUN ln -s /byzcoin .local/share/conode
RUN ln -s /byzcoin .config/conode
COPY byzcoin full bcadmin scmgr byzcoin.sh /root/
LABEL com.centurylinklabs.watchtower.enable="true"

CMD ["./byzcoin.sh"]
