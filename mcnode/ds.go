package main

import (
	mc "github.com/mediachain/concat/mc"
	rocksdb "github.com/tecbot/gorocksdb"
	"path"
	"runtime"
)

type RocksDS struct {
	db *rocksdb.DB
	ro *rocksdb.ReadOptions
	wo *rocksdb.WriteOptions
}

func (ds *RocksDS) Open(home string) error {
	dbpath := path.Join(home, "data")
	// options
	opts := rocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	// bloom filter
	filter := rocksdb.NewBloomFilter(10)
	bbto := rocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetFilterPolicy(filter)
	opts.SetBlockBasedTableFactory(bbto)

	// paraellism tuning
	opts.IncreaseParallelism(runtime.NumCPU())

	// We are not going to be iterating over the datastore as far as I can tell
	// so we can use this option.
	// I don't know if 16MB is a good value for the block cache size
	opts.OptimizeForPointLookup(16)

	db, err := rocksdb.OpenDb(opts, dbpath)
	if err != nil {
		return err
	}

	ds.db = db
	ds.ro = rocksdb.NewDefaultReadOptions()
	ds.wo = rocksdb.NewDefaultWriteOptions()

	return nil
}

func (ds *RocksDS) Put(data []byte) (Key, error) {
	key := mc.Hash(data)
	err := ds.db.Put(ds.wo, key[2:], data)
	return Key(key), err
}

func (ds *RocksDS) Has(key Key) (bool, error) {
	// gorocksdb has no native key check, so we need to do a get
	// small optimization: use Get instead of GetBytes to avoid the extra copy
	// to byte slice when the data is present
	val, err := ds.db.Get(ds.ro, key[2:])
	if err != nil {
		return false, err
	}
	defer val.Free()
	return val.Size() > 0, nil
}

func (ds *RocksDS) Get(key Key) ([]byte, error) {
	return ds.db.GetBytes(ds.ro, key[2:])
}

func (ds *RocksDS) Delete(key Key) error {
	return ds.db.Delete(ds.wo, key[2:])
}

func (ds *RocksDS) Close() {
	ds.db.Close()
}
