package main

import (
	"errors"
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
	return nil, errors.New("Implement me!")
}

func (ds *RocksDS) Has(Key) (bool, error) {
	return false, errors.New("Implement me!")
}

func (ds *RocksDS) Get(Key) ([]byte, error) {
	return nil, errors.New("Implement me!")
}

func (ds *RocksDS) Delete(Key) error {
	return errors.New("Implement me!")
}

func (ds *RocksDS) Close() error {
	return errors.New("Implement me!")
}
