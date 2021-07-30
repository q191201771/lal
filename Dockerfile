FROM golang:1.16.4-buster as builder
WORKDIR /go/src/github.com/q191201771/lal
#ENV GOPROXY=https://goproxy.cn,direct
COPY . .
RUN make build_for_linux

FROM debian:stretch-slim
COPY --from=builder /go/src/github.com/q191201771/lal/bin /lal/bin
COPY --from=builder /go/src/github.com/q191201771/lal/conf /lal/conf
