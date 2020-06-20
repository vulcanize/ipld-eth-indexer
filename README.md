# Vulcanize DB

[![Go Report Card](https://goreportcard.com/badge/github.com/vulcanize/ipfs-blockchain-watcher)](https://goreportcard.com/report/github.com/vulcanize/ipfs-blockchain-watcher)

> Tool for extracting and indexing blockchain data on PG-IPFS

## Table of Contents
1. [Background](#background)
1. [Architecture](#architecture)
1. [Install](#install)
1. [Usage](#usage)
1. [Contributing](#contributing)
1. [License](#license)

## Background
ipfs-blockchain-watcher is a collection of interfaces that are used to extract, process, and store in Postgres-IPFS
all chain data. The raw data indexed by ipfs-blockchain-watcher serves as the basis for more specific watchers and applications.

Currently the service supports complete processing of all Bitcoin and Ethereum data.

## Architecture
More details on the design of ipfs-blockchain-watcher can be found in [here](./documentation/architecture.md)

## Install
1. [Postgres](#postgres)
1. [Goose](#goose)
1. [IPFS](#ipfs)
1. [Blockchain](#blockchain)
1. [Watcher](#watcher)

### Postgres
1. [Install Postgres](https://wiki.postgresql.org/wiki/Detailed_installation_guides)
1. Create a superuser for yourself and make sure `psql --list` works without prompting for a password.
1. `createdb vulcanize_public`
1. `cd $GOPATH/src/github.com/vulcanize/ipfs-blockchain-watcher`
1.  Run the migrations: `make migrate HOST_NAME=localhost NAME=vulcanize_public PORT=5432`
    - There are optional vars `USER=username` and `PASS=password` if the database user is not the default user `postgres` and/or a password is present
    - To rollback a single step: `make rollback NAME=vulcanize_public`
    - To rollback to a certain migration: `make rollback_to MIGRATION=n NAME=vulcanize_public`
    - To see status of migrations: `make migration_status NAME=vulcanize_public`

    * See below for configuring additional environments
    
In some cases (such as recent Ubuntu systems), it may be necessary to overcome failures of password authentication from
localhost. To allow access on Ubuntu, set localhost connections via hostname, ipv4, and ipv6 from peer/md5 to trust in: /etc/postgresql/<version>/pg_hba.conf

(It should be noted that trusted auth should only be enabled on systems without sensitive data in them: development and local test databases)

### Goose
We use [goose](https://github.com/pressly/goose) as our migration management tool. While it is not necessary to use `goose` for manual setup, it
is required for running the automated tests.

### IPFS
We use IPFS to store IPLD objects for each type of data we extract from on chain.

To start, download and install [IPFS](https://github.com/vulcanize/go-ipfs):

`go get github.com/ipfs/go-ipfs`

`cd $GOPATH/src/github.com/ipfs/go-ipfs`

`make install`

If we want to use Postgres as our backing datastore, we need to use the vulcanize fork of go-ipfs.

Start by adding the fork and switching over to it:

`git remote add vulcanize https://github.com/vulcanize/go-ipfs.git`

`git fetch vulcanize`

`git checkout -b postgres_update vulcanize/postgres_update`

Now install this fork of ipfs, first be sure to remove any previous installation:

`make install`

Check that is installed properly by running:

`ipfs`

You should see the CLI info/help output.

And now we initialize with the `postgresds` profile.
If ipfs was previously initialized we will need to remove the old profile first.
We also need to provide env variables for the postgres connection:

We can either set these manually, e.g.
```bash
export IPFS_PGHOST=
export IPFS_PGUSER=
export IPFS_PGDATABASE=
export IPFS_PGPORT=
export IPFS_PGPASSWORD=
```

And then run the ipfs command:

`ipfs init --profile=postgresds`

Or we can use the pre-made script at `GOPATH/src/github.com/ipfs/go-ipfs/misc/utility/ipfs_postgres.sh`
which has usage:

`./ipfs_postgres.sh <IPFS_PGHOST> <IPFS_PGPORT> <IPFS_PGUSER> <IPFS_PGDATABASE>"`

and will ask us to enter the password, avoiding storing it to an ENV variable.

Once we have initialized ipfs, that is all we need to do with it- we do not need to run a daemon during the subsequent processes.

### Blockchain
This section describes how to setup an Ethereum or Bitcoin node to serve as a data source for ipfs-blockchain-watcher

#### Ethereum
For Ethereum, we currently *require* [a special fork of go-ethereum](https://github.com/vulcanize/go-ethereum/tree/statediff_at_anyblock-1.9.11). This can be setup as follows.
Skip this steps if you already have access to a node that displays the statediffing endpoints.

Begin by downloading geth and switching to the vulcanize/rpc_statediffing branch:

`go get github.com/ethereum/go-ethereum`

`cd $GOPATH/src/github.com/ethereum/go-ethereum`

`git remote add vulcanize https://github.com/vulcanize/go-ethereum.git`

`git fetch vulcanize`

`git checkout -b statediffing vulcanize/statediff_at_anyblock-1.9.11`

Now, install this fork of geth (make sure any old versions have been uninstalled/binaries removed first):

`make geth`

And run the output binary with statediffing turned on:

`cd $GOPATH/src/github.com/ethereum/go-ethereum/build/bin`

`./geth --statediff --statediff.streamblock --ws --syncmode=full`

Note: if you wish to access historical data (perform `backFill`) then the node will need to operate as an archival node (`--gcmode=archive`)

Note: other CLI options- statediff specific ones included- can be explored with `./geth help`

The output from geth should mention that it is `Starting statediff service` and block synchronization should begin shortly thereafter.
Note that until it receives a subscriber, the statediffing process does nothing but wait for one. Once a subscription is received, this
will be indicated in the output and node will begin processing and sending statediffs.

Also in the output will be the endpoints that we will use to interface with the node.
The default ws url is "127.0.0.1:8546" and the default http url is "127.0.0.1:8545".
These values will be used as the `ethereum.wsPath` and `ethereum.httpPath` in the config, respectively.

#### Bitcoin
For Bitcoin, ipfs-blockchain-watcher is able to operate entirely through the universally exposed JSON-RPC interfaces.
This means we can use any of the standard full nodes (e.g. bitcoind, btcd) as our data source.

Point at a remote node or set one up locally using the instructions for [bitcoind](https://github.com/bitcoin/bitcoin) and [btcd](https://github.com/btcsuite/btcd).

The default http url is "127.0.0.1:8332". We will use the http endpoint as both the `bitcoin.wsPath` and `bitcoin.httpPath`
(bitcoind does not support websocket endpoints, we are currently using a "subscription" wrapper around the http endpoints)

### Watcher
Finally, we can setup the watcher process itself.

Start by downloading vulcanizedb and moving into the repo:

`go get github.com/vulcanize/ipfs-chain-watcher`

`cd $GOPATH/src/github.com/vulcanize/ipfs-chain-watcher`

Then, build the binary:

`make build`

## Usage
After building the binary, run as

`./ipfs-blockchain-watcher watch --config=<config_file.toml`

### Configuration

Below is the set of universal config parameters for the ipfs-blockchain-watcher command, in .toml form, with the respective environmental variables commented to the side.
This set of parameters needs to be set no matter the chain type.

```toml
[database]
    name     = "vulcanize_public" # $DATABASE_NAME
    hostname = "localhost" # $DATABASE_HOSTNAME
    port     = 5432 # $DATABASE_PORT
    user     = "vdbm" # $DATABASE_USER
    password = "" # $DATABASE_PASSWORD

[ipfs]
    path = "~/.ipfs" # $IPFS_PATH
    mode = "postgres" # $IPFS_MODE

[superNode]
    chain = "bitcoin" # $SUPERNODE_CHAIN
    server = true # $SUPERNODE_SERVER
    ipcPath = "~/.vulcanize/vulcanize.ipc" # $SUPERNODE_IPC_PATH
    wsPath = "127.0.0.1:8082" # $SUPERNODE_WS_PATH
    httpPath = "127.0.0.1:8083" # $SUPERNODE_HTTP_PATH
    sync = true # $SUPERNODE_SYNC
    workers = 1 # $SUPERNODE_WORKERS
    backFill = true # $SUPERNODE_BACKFILL
    frequency = 45 # $SUPERNODE_FREQUENCY
    batchSize = 1 # $SUPERNODE_BATCH_SIZE
    batchNumber = 50 # $SUPERNODE_BATCH_NUMBER
    timeout = 300 # $HTTP_TIMEOUT
    validationLevel = 1 # $SUPERNODE_VALIDATION_LEVEL
```

Additional parameters need to be set depending on the specific chain.

For Bitcoin:

```toml
[bitcoin]
    wsPath  = "127.0.0.1:8332" # $BTC_WS_PATH
    httpPath = "127.0.0.1:8332" # $BTC_HTTP_PATH
    pass = "password" # $BTC_NODE_PASSWORD
    user = "username" # $BTC_NODE_USER
    nodeID = "ocd0" # $BTC_NODE_ID
    clientName = "Omnicore" # $BTC_CLIENT_NAME
    genesisBlock = "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f" # $BTC_GENESIS_BLOCK
    networkID = "0xD9B4BEF9" # $BTC_NETWORK_ID
```

For Ethereum:

```toml
[ethereum]
    wsPath  = "127.0.0.1:8546" # $ETH_WS_PATH
    httpPath = "127.0.0.1:8545" # $ETH_HTTP_PATH
    nodeID = "arch1" # $ETH_NODE_ID
    clientName = "Geth" # $ETH_CLIENT_NAME
    genesisBlock = "0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3" # $ETH_GENESIS_BLOCK
    networkID = "1" # $ETH_NETWORK_ID
```

### Exposing the data
We can expose a number of different APIs for remote access to ipfs-blockchain-watcher data, these are dicussed in more detail [here](./documentation/apis.md)

### Testing
`make test` will run the unit tests
`make test` setups a clean `vulcanize_testing` db

## Contributing
Contributions are welcome!

VulcanizeDB follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/1/4/code-of-conduct).

## License
[AGPL-3.0](LICENSE) © Vulcanize Inc