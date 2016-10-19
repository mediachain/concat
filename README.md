# concat

_concat<sup>[1](#footnote-1)</sup>: concatenating cat, by contrast to [copycat](https://github.com/atomix/copycat), used in a [previous prototype](https://github.com/mediachain/oldchain)_

**concat** is a set of daemons that provide the backbone of the Mediachain system. Please see [RFC 4](https://github.com/mediachain/mediachain/blob/master/rfc/mediachain-rfc-4.md), [concat.md](https://github.com/mediachain/mediachain/blob/master/docs/concat.md) and the [9/15/16](https://blog.mediachain.io/looking-backwards-looking-forwards-9149bf00f876#.c4xrhcdwj) developer update for a high level overview of this design.

The two principal daemons are **mcdir**, which provides a service for peer registration and discovery, and **mcnode**, which actually stores, accepts and returns data.

## mcdir
See also [roles](https://github.com/mediachain/mediachain/blob/master/rfc/mediachain-rfc-4-roles.md#directory-servers).

### API
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
* `POST /delete` -- delete statements matching this MCQL DELETE query                                                         * `POST /data/put` -- add a batch of data objects to datastore
* `GET /data/get/{objectId}` -- get an object from the datastore
* `GET /status` -- get node network state
* `POST /status/{state}` -- control network state (online/offline/public)
* `GET/POST /config/dir` -- retrieve/set the configured directory
* `GET/POST /config/nat` -- retrieve/set NAT setting                                                                           * `GET/POST /config/info` -- retrieve/set info string
* `GET /dir/list` -- list known peers
* `GET /net/addr` -- list known addresses

### MCQL
A limited SQL-like language is provided to query and delete statements and system metadata. Some possible queries:

```sql
SELECT namespace FROM *
```
```sql
SELECT COUNT(*) FROM *
```
```sql
SELECT (id, timestamp) FROM foo.bar -- foo.bar here is a namespace
```

Note that full relational algebra semantics are not (yet) supported and more complex queries may yield unexpected results.

## Installation
After cloning to the appropriate GOPATH,
```sh
./setup.sh
./build.sh
```

## Usage
```sh
# start directory
shell0 $ mcdir
2016/09/09 21:20:02 Generating key pair
2016/09/09 21:20:02 ID: QmbF87NnyoN3msnmifXhCMUpCRK6C8uXxqAgr93QVa67xS
2016/09/09 21:20:02 I am /ip4/127.0.0.1/tcp/9000/QmbF87NnyoN3msnmifXhCMUpCRK6C8uXxqAgr93QVa67xS`
...

# start and register node 1
shell1 $ mcnode /ip4/127.0.0.1/tcp/9000/QmbF87NnyoN3msnmifXhCMUpCRK6C8uXxqAgr93QVa67xS
2016/09/09 21:20:30 Generating key pair
2016/09/09 21:20:30 ID: QmXdkCFvS4EzuSF3XeaWAS5HxM2uPC9TKWcTZGoiBERobU
2016/09/09 21:20:30 I am /ip4/127.0.0.1/tcp/9001/QmXdkCFvS4EzuSF3XeaWAS5HxM2uPC9TKWcTZGoiBERobU
2016/09/09 21:20:30 Serving client interface at 127.0.0.1:9002
2016/09/09 21:20:30 Registering with directory
...

# start and register node 2
shell2 $ mcnode -l 9003 -c 9004 /ip4/127.0.0.1/tcp/9000/QmbF87NnyoN3msnmifXhCMUpCRK6C8uXxqAgr93QVa67xS
2016/09/09 21:21:13 Generating key pair
2016/09/09 21:21:13 ID: QmQg6PiJ6ouBK6pEukZ9rsVRJC7gnt8gvp78gp7XtippJ6
2016/09/09 21:21:13 I am /ip4/127.0.0.1/tcp/9003/QmQg6PiJ6ouBK6pEukZ9rsVRJC7gnt8gvp78gp7XtippJ6
2016/09/09 21:21:13 Serving client interface at 127.0.0.1:9004
2016/09/09 21:21:13 Registering with directory
...

# ping
shell3 $ curl http://127.0.0.1:9002/ping/QmQg6PiJ6ouBK6pEukZ9rsVRJC7gnt8gvp78gp7XtippJ6
OK

# publish a statement
shell3 $ curl -H "Content-Type: application/json" -d '{"object": "QmABC", "refs": ["abc"], "tags": ["test"]}' http://127.0.0.1:9002/publish/foo.bar
QmWwnVop4hHB3K6z2vuUgU9rh5VrDwMnwuiP9LZew7NzUK:1473848347:0
shell3 $ ls /tmp/mcnode/stmt/
QmWwnVop4hHB3K6z2vuUgU9rh5VrDwMnwuiP9LZew7NzUK:1473848347:0
shell3 $ curl http://127.0.0.1:9002/stmt/QmWwnVop4hHB3K6z2vuUgU9rh5VrDwMnwuiP9LZew7NzUK:1473848347:0
{"id":"QmWwnVop4hHB3K6z2vuUgU9rh5VrDwMnwuiP9LZew7NzUK:1473848347:0","publisher":"QmWwnVop4hHB3K6z2vuUgU9rh5VrDwMnwuiP9LZew7NzUK","namespace":"foo.bar","Body":{"Simple":{"object":"QmABC","refs":["abc"],"tags":["test"]}}}

# save some arbitrary metadata
shell3 $ curl -s -H "Content-Type: application/json" -d '{"data": "FCG389RpiSmWkbK96fd0if4gmhw="}' http://127.0.0.1:9002/data/put
QmT7KTeJYJ7pnvsUk8g56AqFgp1Cqk91HnuhyhmaYZW4dr

# read it back
shell3 $ curl http://127.0.0.1:9002/data/get/QmT7KTeJYJ7pnvsUk8g56AqFgp1Cqk91HnuhyhmaYZW4dr
{"data":"FCG389RpiSmWkbK96fd0if4gmhw="}

# merge etc
```


1. <a name="footnote-1"></a> alternately [funes](http://www4.ncsu.edu/~jjsakon/FunestheMemorious.pdf), cf. [aleph](https://github.com/mediachain/aleph)
