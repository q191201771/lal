# Build
FROM golang:1.16.4-buster as builder
WORKDIR /go/src/github.com/q191201771/lal
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
COPY . .
RUN make build_for_linux

# Output
FROM debian:stretch-slim

EXPOSE 1935 8080 4433 5544 8083 8084 30000-30100/udp

COPY --from=builder /go/src/github.com/q191201771/lal/bin/lalserver /lal/bin/lalserver
COPY --from=builder /go/src/github.com/q191201771/lal/conf/lalserver.conf.json /lal/conf/lalserver.conf.json
COPY --from=builder /go/src/github.com/q191201771/lal/conf/cert.pem /lal/conf/cert.pem
COPY --from=builder /go/src/github.com/q191201771/lal/conf/key.pem /lal/conf/key.pem

WORKDIR /lal
CMD ["sh","-c","./bin/lalserver -c conf/lalserver.conf.json"]
