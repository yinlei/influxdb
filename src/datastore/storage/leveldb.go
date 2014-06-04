package storage

import (
	"bytes"

	levigo "github.com/jmhodges/levigo"
)

const LEVELDB_NAME = "leveldb"

func init() {
	registerEngine(LEVELDB_NAME, NewLevelDb)
}

type LevelDB struct {
	db    *levigo.DB
	opts  *levigo.Options
	wopts *levigo.WriteOptions
	ropts *levigo.ReadOptions
	cache *levigo.Cache
	path  string
}

func NewLevelDb(path string) (Engine, error) {
	opts := levigo.NewOptions()
	cache := levigo.NewLRUCache(100 * 1024 * 1024)
	opts.SetCompression(levigo.NoCompression)
	opts.SetCache(cache)
	opts.SetCreateIfMissing(true)
	db, err := levigo.Open(path, opts)
	wopts := levigo.NewWriteOptions()
	ropts := levigo.NewReadOptions()
	return LevelDB{db, opts, wopts, ropts, cache, path}, err
}

func (db LevelDB) Compact() {
	db.db.CompactRange(levigo.Range{})
}

func (db LevelDB) Close() {
	db.cache.Close()
	db.ropts.Close()
	db.wopts.Close()
	db.opts.Close()
	db.db.Close()
}

func (db LevelDB) Put(key, value []byte) error {
	return db.BatchPut([]Write{Write{key, value}})
}

func (db LevelDB) Get(key []byte) ([]byte, error) {
	return db.db.Get(db.ropts, key)
}

func (_ LevelDB) Name() string {
	return LEVELDB_NAME
}

func (db LevelDB) Path() string {
	return db.path
}

func (db LevelDB) BatchPut(writes []Write) error {
	wb := levigo.NewWriteBatch()
	defer wb.Close()
	for _, w := range writes {
		if w.Value == nil {
			wb.Delete(w.Key)
			continue
		}
		wb.Put(w.Key, w.Value)
	}
	return db.db.Write(db.wopts, wb)
}

func (db LevelDB) Del(start, finish []byte) error {
	wb := levigo.NewWriteBatch()
	defer wb.Close()

	itr := db.Iterator()
	defer itr.Close()

	for itr.Seek(start); itr.Valid(); itr.Next() {
		k := itr.Key()
		if bytes.Compare(k, finish) > 0 {
			break
		}
		wb.Delete(k)
	}

	if err := itr.Error(); err != nil {
		return err
	}

	return db.db.Write(db.wopts, wb)
}

type LevelDbIterator struct {
	_itr *levigo.Iterator
	err  error
}

func (itr *LevelDbIterator) Seek(key []byte) {
	itr._itr.Seek(key)
}

func (itr *LevelDbIterator) Next() {
	itr._itr.Next()
}

func (itr *LevelDbIterator) Prev() {
	itr._itr.Prev()
}

func (itr *LevelDbIterator) Valid() bool {
	return itr._itr.Valid()
}

func (itr *LevelDbIterator) Error() error {
	return itr.err
}

func (itr *LevelDbIterator) Key() []byte {
	return itr._itr.Key()
}

func (itr *LevelDbIterator) Value() []byte {
	return itr._itr.Value()
}

func (itr *LevelDbIterator) Close() error {
	itr._itr.Close()
	return nil
}

func (db LevelDB) Iterator() Iterator {
	itr := db.db.NewIterator(db.ropts)

	return &LevelDbIterator{itr, nil}
}
