FROM debian:stretch-slim
RUN apt update && apt install -y procps ca-certificates && apt clean
WORKDIR /root/
RUN mkdir /byzcoin
RUN mkdir -p .local/share .config
RUN ln -s /byzcoin .local/share/conode
RUN ln -s /byzcoin .config/conode
COPY byzcoin byzcoin.sh /root/

CMD ["./byzcoin.sh"]
