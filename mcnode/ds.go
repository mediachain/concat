package main

import (
	"context"
	mc "github.com/mediachain/concat/mc"
	rocksdb "github.com/tecbot/gorocksdb"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
)

type RocksDS struct {
	db *rocksdb.DB
	ro *rocksdb.ReadOptions
	wo *rocksdb.WriteOptions
	fo *rocksdb.FlushOptions
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
	// so we can use OptimizeForPointLookup
	// I don't know what a good value for the blockcache size is, so configure
	// from ENV, with a default of 64MB
	bcmb := 64
	ev := os.Getenv("MCBLOCKCACHESZ")
	if ev != "" {
		mb, err := strconv.Atoi(ev)
		if err == nil && mb > 0 {
			log.Printf("Using %dMB for datastore block cache", mb)
			bcmb = mb
		} else {
			log.Printf("Warning: Ignoring bad block cache size: %s", ev)
		}
	}
	opts.OptimizeForPointLookup(uint64(bcmb))

	db, err := rocksdb.OpenDb(opts, dbpath)
	if err != nil {
		return err
	}

	ds.db = db
	ds.ro = rocksdb.NewDefaultReadOptions()
	ds.wo = rocksdb.NewDefaultWriteOptions()
	ds.fo = rocksdb.NewDefaultFlushOptions()

	return nil
}

func (ds *RocksDS) Put(data []byte) (Key, error) {
	key := mc.Hash(data)
	err := ds.db.Put(ds.wo, key[2:], data)
	return Key(key), err
}

func (ds *RocksDS) PutBatch(batch [][]byte) ([]Key, error) {
	keys := make([]Key, len(batch))
	wb := rocksdb.NewWriteBatch()
	defer wb.Destroy()

	for x, data := range batch {
		key := mc.Hash(data)
		wb.Put(key[2:], data)
		keys[x] = Key(key)
	}

	err := ds.db.Write(ds.wo, wb)
	if err != nil {
		return nil, err
	}

	return keys, nil
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

func (ds *RocksDS) Sync() error {
	return ds.db.Flush(ds.fo)
}

func (ds *RocksDS) IterKeys(ctx context.Context) (<-chan Key, error) {
	err := ds.Sync()
	if err != nil {
		return nil, err
	}

	ch := make(chan Key)
	go func() {
		defer close(ch)

		it := ds.db.NewIterator(ds.ro)
		defer it.Close()

	loop:
		for it.SeekToFirst(); it.Valid(); it.Next() {
			kslice := it.Key()
			key := mc.HashFromBytes(kslice.Data())
			kslice.Free()
			select {
			case ch <- Key(key):
			case <-ctx.Done():
				break loop
			}
		}
	}()
	return ch, nil
}

func (ds *RocksDS) Compact() {
	ds.db.CompactRange(rocksdb.Range{})
}

func (ds *RocksDS) Close() {
	ds.db.Close()
}
