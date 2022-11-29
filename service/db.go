package service

import (
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type LevelDB struct {
	*leveldb.DB
	wo *opt.WriteOptions
	ro *opt.ReadOptions
}

func NewLevelDbStore(path string, wo *opt.WriteOptions, ro *opt.ReadOptions) *LevelDB {
	blocks, err := leveldb.OpenFile(path, nil)
	if err != nil {
		panic(fmt.Errorf("failed to open db file (%v): %v", path, err))
	}

	return &LevelDB{blocks, wo, ro}
}

func (db *LevelDB) Close() error {
	return db.DB.Close()
}

func (db *LevelDB) Put(key, value []byte) error {
	return db.DB.Put(key, value, db.wo)
}

func (db *LevelDB) Has(key []byte) (bool, error) {
	return db.DB.Has(key, db.ro)
}

func (db *LevelDB) Get(key []byte) (value []byte, err error) {
	return db.DB.Get(key, db.ro)
}

func (db *LevelDB) Delete(key []byte) error {
	return db.DB.Delete(key, db.wo)
}

func (db *LevelDB) Iterator() iterator.Iterator {
	return db.DB.NewIterator(nil, db.ro)
}
