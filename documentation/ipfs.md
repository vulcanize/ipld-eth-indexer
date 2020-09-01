### PG-IPFS configuration

This doc walks through the steps to install IPFS and configure it to use Postgres (public.blocks table) as its backing kv datastore.

1. Start by downloading and moving into the IPFS repo:

`go get github.com/ipfs/go-ipfs`

`cd $GOPATH/src/github.com/ipfs/go-ipfs`

2. Add the [Postgres-supporting fork](https://github.com/vulcanize/go-ipfs) and switch over to it:

`git remote add vulcanize https://github.com/vulcanize/go-ipfs.git`

`git fetch vulcanize`

`git checkout -b postgres_update tags/v0.4.22-alpha`

3. Now install this fork of ipfs, first be sure to remove any previous installation:

`make install`

4. Check that is installed properly by running:

`ipfs`

You should see the CLI info/help output.

5. Now we initialize with the `postgresds` profile.
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