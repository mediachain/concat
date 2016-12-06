#!/bin/bash

die() {
    echo "FAILED"
    exit 1
}

echo "Installing gx"
go get github.com/whyrusleeping/gx github.com/whyrusleeping/gx-go || die

echo "Installing libp2p deps"
gx --verbose install  || die

echo "Installing unvendored deps"
go get github.com/gorilla/mux github.com/mattn/go-sqlite3 github.com/mitchellh/go-homedir || die

echo "Installing gorocksdb; this can take a while!"
go get -tags=embed github.com/mediachain/gorocksdb || die

echo "DONE"
