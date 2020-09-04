FROM golang:1.13-alpine as builder

RUN apk --update --no-cache add make git g++ linux-headers
# DEBUG
RUN apk add busybox-extras

# Get and build ipld-eth-indexer
ADD . /go/src/github.com/vulcanize/ipld-eth-indexer
WORKDIR /go/src/github.com/vulcanize/ipld-eth-indexer
RUN GO111MODULE=on GCO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o ipld-eth-indexer .

# Build migration tool
WORKDIR /
RUN go get -u -d github.com/pressly/goose/cmd/goose
WORKDIR /go/src/github.com/pressly/goose/cmd/goose
RUN GCO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -tags='no_mysql no_sqlite' -o goose .

WORKDIR /go/src/github.com/vulcanize/ipld-eth-indexer

# app container
FROM alpine

ARG USER="vdm"
ARG CONFIG_FILE="./environments/example.toml"

RUN adduser -Du 5000 $USER
WORKDIR /app
RUN chown $USER /app
USER $USER

# chown first so dir is writable
# note: using $USER is merged, but not in the stable release yet
COPY --chown=5000:5000 --from=builder /go/src/github.com/vulcanize/ipld-eth-indexer/$CONFIG_FILE config.toml
COPY --chown=5000:5000 --from=builder /go/src/github.com/vulcanize/ipld-eth-indexer/startup_script.sh .

# keep binaries immutable
COPY --from=builder /go/src/github.com/vulcanize/ipld-eth-indexer/ipld-eth-indexer ipld-eth-indexer
COPY --from=builder /go/src/github.com/pressly/goose/cmd/goose/goose goose
COPY --from=builder /go/src/github.com/vulcanize/ipld-eth-indexer/db/migrations migrations/vulcanizedb
COPY --from=builder /go/src/github.com/vulcanize/ipld-eth-indexer/environments environments

EXPOSE 8080
EXPOSE 8081

ENTRYPOINT ["/app/startup_script.sh"]
