# Concat

_concat<sup>[1](#footnote-1)</sup>: concatenating cat, by contrast to [copycat](https://github.com/atomix/copycat), used in a [previous prototype](https://github.com/mediachain/oldchain)_

**concat** is a set of daemons that provide the backbone of the Mediachain peer-to-peer network.
Please see [RFC 4](https://github.com/mediachain/mediachain/blob/master/rfc/mediachain-rfc-4.md), [concat.md](https://github.com/mediachain/mediachain/blob/master/docs/concat.md) and the [9/15/16](https://blog.mediachain.io/looking-backwards-looking-forwards-9149bf00f876#.c4xrhcdwj) developer update for a high level overview of this design.

The two main programs in concat is **mcnode**, which is the implementation of a fully featured Mediachain node, and **mcdir**, which implements a directory server for facilitating node discovery and connectivity.

## Installation
Concat requires Go 1.7 or later.

First clone the repo in your `$GOPATH`:
```
$ mkdir -p $GOPATH/src/github.com/mediachain
$ git clone https://github.com/mediachain/concat.git $GOPATH/src/github.com/mediachain/concat
```

You can then run the setup and install scripts which will fetch dependencies and install the programs in `$GOPATH/bin`:
```
$ cd $GOPATH/src/github.com/mediachain/concat
$ ./setup.sh && ./install.sh 
```

## Usage

TODO

## mcnode
### Architecture
The node contains the **statment db** (including **envelopes**, **refs**, etc) and the **datastore**. The datastore contains the metadata _per se_, as CBOR objects ([IPLD](https://github.com/ipld/specs/tree/master/ipld) compatible to the best of our ability) of unspecified schema, stored in RocksDB in point lookup mode.

The statement db contains **statements** about one (currently) or more metadata objects: their publisher, namespace, timestamp and signature. Statements are [protobuf objects](https://github.com/mediachain/concat/blob/master/proto/stmt.proto) sent over the wire between peers to signal publication or sharing of metadata; when stored, they act as an index to the datastore. This db is currently stored in SQLite.

### libp2p API
TODO

### REST API
A REST API is provided for controlling the node. This is an administrative interface and should NOT be accessible to the wider network.

* `GET /id` -- node info for this peer
* `GET /id/{peerId}` -- node info for peer given by peerId
* `GET /ping/{peerId}` -- ping!
* `POST /publish/{namespace}` -- publish a batch of statements to the specified namespace 
* `GET /stmt/{statementId}` -- retrieve statement by statementId
* `POST /query` -- issue MCQL SELECT query on this peer
* `POST /query/{peerId}` -- issue MCQL SELECT query on a remote peer
* `POST /merge/{peerId}` -- query a remote peer and merge the resulting statements into this one
* `POST /delete` -- delete statements matching this MCQL DELETE query
* `POST /data/put` -- add a batch of data objects to datastore
* `GET /data/get/{objectId}` -- get an object from the datastore
* `GET /status` -- get node network state
* `POST /status/{state}` -- control network state (online/offline/public)
* `GET/POST /config/dir` -- retrieve/set the configured directory
* `GET/POST /config/nat` -- retrieve/set NAT setting                                                                           * `GET/POST /config/info` -- retrieve/set info string
* `GET /dir/list` -- list known peers
* `GET /net/addr` -- list known addresses

### MCQL
MCQL is a query-language for retrieving statements from the node's statement db.
It supports `SELECT` and `DELETE` statements with a syntax very similar to SQL, where
namespaces play the role of tables.

Some basic example statements:

```sql
SELECT namespace FROM *
```
```sql
SELECT COUNT(*) FROM *
```
```sql
SELECT (id, timestamp) FROM foo.bar -- foo.bar here is a namespace
```

The full grammar for MCQL is defined as a PEG in [query.peg](mc/query/query.peg)

## mcdir
See also [roles](https://github.com/mediachain/mediachain/blob/master/rfc/mediachain-rfc-4-roles.md#directory-servers).

### API
TODO


1. <a name="footnote-1"></a> alternately [funes](http://www4.ncsu.edu/~jjsakon/FunestheMemorious.pdf), cf. [aleph](https://github.com/mediachain/aleph)
