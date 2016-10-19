# concat

_concat<sup>[1](#footnote-1)</sup>: concatenating cat, by contrast to [copycat](https://github.com/atomix/copycat), used in a [previous prototype](https://github.com/mediachain/oldchain)_

**concat** is a set of daemons that provide the backbone of the Mediachain system. Please see [RFC 4](https://github.com/mediachain/mediachain/blob/master/rfc/mediachain-rfc-4.md), [concat.md](https://github.com/mediachain/mediachain/blob/master/docs/concat.md) and the [9/15/16](https://blog.mediachain.io/looking-backwards-looking-forwards-9149bf00f876#.c4xrhcdwj) developer update for a high level overview of this design.

The two principal daemons are **mcdir**, which provides a service for peer registration and discovery, and **mcnode**, which actually stores, accepts and returns data.

## mcdir
See also [roles](https://github.com/mediachain/mediachain/blob/master/rfc/mediachain-rfc-4-roles.md#directory-servers).

### API
TODO

## mcnode

### libp2p API
TODO

### REST API
TODO

## Installing
TODO

## Usage
TODO

1. <a name="footnote-1"></a> alternately [funes](http://www4.ncsu.edu/~jjsakon/FunestheMemorious.pdf), cf. [aleph](https://github.com/mediachain/aleph)
