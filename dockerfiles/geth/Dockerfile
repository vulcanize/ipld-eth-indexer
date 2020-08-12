FROM golang:1.13-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git
# DEBUG
RUN apk add busybox-extras

# Get and build vulcanize's geth fork
RUN go get -d github.com/ethereum/go-ethereum
WORKDIR /go/src/github.com/ethereum/go-ethereum
RUN git remote add vulcanize https://github.com/vulcanize/go-ethereum.git
RUN git fetch vulcanize
RUN git checkout -b statediff_geth vulcanize/statediff_at_anyblock-1.9.11
RUN make geth

# app container
FROM alpine

RUN apk add --no-cache ca-certificates
WORKDIR /app

# keep binaries immutable
COPY --from=builder /go/src/github.com/ethereum/go-ethereum/build/bin/geth geth

EXPOSE 8545 8546 8547 30303 30303/udp

ENTRYPOINT ["geth"]
