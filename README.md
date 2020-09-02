# ipld-eth-indexer

[![Go Report Card](https://goreportcard.com/badge/github.com/vulcanize/ipld-eth-indexer)](https://goreportcard.com/report/github.com/vulcanize/ipld-eth-indexer)

>  ipld-eth-indexer is used to extract, transform, and load all Ethereum IPLD data into an IPFS-backing Postgres datastore while generating useful secondary indexes around the data in other Postgres tables

## Table of Contents
1. [Background](#background)
1. [Install](#install)
1. [Usage](#usage)
1. [Contributing](#contributing)
1. [License](#license)

## Background
ipld-eth-indexer is a collection of interfaces that are used to extract, transform, store, and index
all Ethereum IPLD data in Postgres. The raw data indexed by ipld-eth-indexer serves as the basis for more specific watchers and applications.

## Dependencies
Minimal build dependencies
* Go (1.13)
* Git
* GCC compiler
* This repository

Potential external dependencies
* Goose
* Postgres
* Statediffing go-ethereum

## Install
1. [Goose](#goose)
1. [Postgres](#postgres)
1. [Ethereum](#ethereum)
1. [Indexer](#indexer)

### Goose
[goose](https://github.com/pressly/goose) is used for migration management. While it is not necessary to use `goose` for manual setup, it
is required for running the automated tests and is used by the `make migrate` command.

### Postgres
1. [Install Postgres](https://wiki.postgresql.org/wiki/Detailed_installation_guides)
1. Create a superuser for yourself and make sure `psql --list` works without prompting for a password.
1. `createdb vulcanize_public`
1. `cd $GOPATH/src/github.com/vulcanize/ipld-eth-indexer`
1.  Run the migrations: `make migrate HOST_NAME=localhost NAME=vulcanize_public PORT=5432`
    - There are optional vars `USER=username:password` if the database user is not the default user `postgres` and/or a password is present
    - To rollback a single step: `make rollback NAME=vulcanize_public`
    - To rollback to a certain migration: `make rollback_to MIGRATION=n NAME=vulcanize_public`
    - To see status of migrations: `make migration_status NAME=vulcanize_public`

    * See below for configuring additional environments
    
In some cases (such as recent Ubuntu systems), it may be necessary to overcome failures of password authentication from
localhost. To allow access on Ubuntu, set localhost connections via hostname, ipv4, and ipv6 from peer/md5 to trust in: /etc/postgresql/<version>/pg_hba.conf

(It should be noted that trusted auth should only be enabled on systems without sensitive data in them: development and local test databases)

### Ethereum
[A special fork of go-ethereum](https://github.com/vulcanize/go-ethereum/tree/statediff_at_anyblock-1.9.11) is currently *required*.
This can be setup as follows.
Skip this step if you already have access to a node that displays the statediffing endpoints.

Begin by downloading geth and switching to the statediffing branch:

`GO111MODULE=off go get -d github.com/ethereum/go-ethereum`

`cd $GOPATH/src/github.com/ethereum/go-ethereum`

`git remote add vulcanize https://github.com/vulcanize/go-ethereum.git`

`git fetch vulcanize`

`git checkout -b statediffing vulcanize/statediff_at_anyblock-1.9.11`

Now, install this fork of geth (make sure any old versions have been uninstalled/binaries removed first):

`make geth`

And run the output binary with statediffing turned on:

`cd $GOPATH/src/github.com/ethereum/go-ethereum/build/bin`

`./geth --syncmode=full --statediff --ws`

Note: to access historical data (perform `backFill`) the node will need to operate as an archival node (`--gcmode=archive`) with rpc endpoints
exposed (`--rpc --rpcapi=eth,statediff,net`)

Warning: There is a good chance even a fully synced archive node has incomplete historical state data to some degree

The output from geth should mention that it is `Starting statediff service` and block synchronization should begin shortly thereafter.
Note that until it receives a subscriber, the statediffing process does nothing but wait for one. Once a subscription is received, this
will be indicated in the output and the node will begin processing and sending statediffs.

Also in the output will be the endpoints that will be used to interface with the node.
The default ws url is "127.0.0.1:8546" and the default http url is "127.0.0.1:8545".
These values will be used as the `ethereum.wsPath` and `ethereum.httpPath` in the config, respectively.

### Indexer
Finally, setup the indexer process itself.

Start by downloading ipld-eth-indexer and moving into the repo:

`GO111MODULE=off go get -d github.com/vulcanize/ipld-eth-indexer`

`cd $GOPATH/src/github.com/vulcanize/ipld-eth-indexer`

Then, build the binary:

`make build`

## Usage
After building the binary, three commands are available

* Sync: Streams raw chain data at the head, transforms it into IPLD objects, and indexes the resulting set of CIDs in Postgres with useful metadata.

`./ipld-eth-indexer sync --config=<the name of your config file.toml>`

* Backfill: Automatically searches for and detects gaps in the DB; syncs the data to fill these gaps.

`./ipld-eth-indexer backfill --config=<the name of your config file.toml>`

* Resync: Manually define block ranges within which to (re)fill data over HTTP; can be ran in parallel with non-overlapping regions to scale historical data processing

`./ipld-eth-indexer resync --config=<the name of your config file.toml>`


### Configuration

Below is the set of parameters for the ipld-eth-indexer command, in .toml form, with the respective environmental variables commented to the side.
The corresponding CLI flags can be found with the `./ipld-eth-indexer {command} --help` command.

```toml
[database]
    name     = "vulcanize_public" # $DATABASE_NAME
    hostname = "localhost" # $DATABASE_HOSTNAME
    port     = 5432 # $DATABASE_PORT
    user     = "postgres" # $DATABASE_USER
    password = "" # $DATABASE_PASSWORD

[log]
    level = "info" # $LOGRUS_LEVEL

[sync]
    workers = 4 # $SYNC_WORKERS

[backfill]
    frequency = 15 # $BACKFILL_FREQUENCY
    batchSize = 2 # $BACKFILL_BATCH_SIZE
    workers = 4 # $BACKFILL_WORKERS
    timeout = 300 # $HTTP_TIMEOUT
    validationLevel = 1 # $BACKFILL_VALIDATION_LEVEL

[resync]
    type = "full" # $RESYNC_TYPE
    start = 0 # $RESYNC_START
    stop = 0 # $RESYNC_STOP
    batchSize = 2 # $RESYNC_BATCH_SIZE
    workers = 4 # $RESYNC_WORKERS
    timeout = 300 # $HTTP_TIMEOUT
    clearOldCache = false # $RESYNC_CLEAR_OLD_CACHE
    resetValidation = false # $RESYNC_RESET_VALIDATION

[ethereum]
    wsPath  = "127.0.0.1:8546" # $ETH_WS_PATH
    httpPath = "127.0.0.1:8545" # $ETH_HTTP_PATH
    nodeID = "arch1" # $ETH_NODE_ID
    clientName = "Geth" # $ETH_CLIENT_NAME
    genesisBlock = "0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3" # $ETH_GENESIS_BLOCK
    networkID = "1" # $ETH_NETWORK_ID
    chainID = "1" # $ETH_CHAIN_ID
```

`sync`, `backfill`, and `resync` parameters are only applicable to their respective commands.

`backfill` and `resync` require only an `ethereum.httpPath` while `sync` requires only an `ethereum.wsPath`.

### Exposing the data
* Use [ipld-eth-server](https://github.com/vulcanize/ipld-eth-server) to expose standard eth JSON RPC endpoints as well as unique ones
* Use [Postgraphile](https://www.graphile.org/postgraphile/) to expose GraphQL endpoints on top of the Postgres tables

e.g.

`postgraphile --plugins @graphile/pg-pubsub --subscriptions --simple-subscriptions -c postgres://localhost:5432/vulcanize_public?sslmode=disable -s public,eth -a -j`


This will stand up a Postgraphile server on the public and eth schemas- exposing GraphQL endpoints for all of the tables contained under those schemas.
All of their data can then be queried with standard [GraphQL](https://graphql.org) queries.

* Use PG-IPFS to expose the raw IPLD data. More information on how to stand up an IPFS node on top
of Postgres can be found [here](./documentation/ipfs.md)

### Testing
`make test` will run the unit tests  
`make test` setups a clean `vulcanize_testing` db

## Contributing
Contributions are welcome!

VulcanizeDB follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/1/4/code-of-conduct).

## License
[AGPL-3.0](LICENSE) © Vulcanize Inc
