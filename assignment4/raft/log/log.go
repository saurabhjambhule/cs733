package log

//
// log.Log is an array on disk. It can be appended or truncated (all action is at the end).
// It keeps track of the last index to which data is written (-1 if the log is empty)
//

import (
	"bytes"
	"encoding/gob"
	_ "fmt"
	lru "github.com/hashicorp/golang-lru/simplelru"
	"github.com/syndtr/goleveldb/leveldb"
	"strconv"
	"sync"
)

type LogEntry struct {
	Data interface{}
}

type Log struct {
	sync.Mutex
	cache     *lru.LRU
	ldb       *leveldb.DB
	lastIndex int64 // The highest index that has data.-1 if log is empty

}

var DEFAULT_LRU int = 256 // entries

// Create an indexed log at dbpath, or open one if it exists
func Open(dbpath string) (*Log, error) {
	gob.Register(LogEntry{})
	db, err := leveldb.OpenFile(dbpath, nil)
	if err != nil {
		return nil, err
	}
	var lg = &Log{ldb: db, lastIndex: -1}
	lg.SetCacheSize(DEFAULT_LRU)
	lastIndBytes, err := db.Get([]byte("lastIndex"), nil)
	if err == nil {
		lg.lastIndex = toIndex(lastIndBytes)
	}
	return lg, nil
}

func (lg *Log) SetCacheSize(size int) {
	lg.cache, _ = lru.NewLRU(size, nil)
}

// Append at the next available index, which is log.GetLastIndex()+1
func (lg *Log) Append(data interface{}) error {
	lg.setLastIndex(lg.lastIndex + 1)

	tr, err := lg.ldb.OpenTransaction()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tr.Discard()
		} else {
			tr.Commit()
		}
	}()

	lg.cache.Add(lg.lastIndex, data)

	ibytes := toBytes(lg.lastIndex)
	dbytes, err := encode(data)
	if err != nil {
		return err
	}

	if err := tr.Put(ibytes, dbytes, nil); err != nil {
		return err
	}

	if err := tr.Put([]byte("lastIndex"), ibytes, nil); err != nil {
		return err
	}
	return err
}

func (lg *Log) RegisterSampleEntry(e interface{}) {
	gob.Register(e)
}

// Remove all entries from (and including) fromIndex to the end
// GetLastIndex() will return fromIndex - 1
func (lg *Log) TruncateToEnd(fromIndex int64) error {
	tr, err := lg.ldb.OpenTransaction()
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = tr.Commit()
		} else {
			tr.Discard()
		}
	}()

	for i := fromIndex; i <= lg.lastIndex; i++ {
		lg.cache.Remove(i)
		err = tr.Delete(toBytes(i), nil)
		if err != nil {
			return err
		}
	}

	ibytes := toBytes(fromIndex - 1)
	if err := tr.Put([]byte("lastIndex"), ibytes, nil); err != nil {
		return err
	}
	lg.setLastIndex(fromIndex - 1)
	return nil
}

// Get data at index
func (lg *Log) Get(index int64) (interface{}, error) {
	var data interface{}
	var err error
	var dbytes []byte
	var ok bool
	if data, ok = lg.cache.Get(index); ok {
		return data, nil
	}
	if dbytes, err = lg.ldb.Get(toBytes(index), nil); err == nil {
		data, err = decode(dbytes)
		if err != nil {
			lg.cache.Add(index, data)
		}
	}

	return data, err
}

func (lg *Log) Close() {
	if lg.ldb != nil {
		lg.ldb.Close()
	}
}

func (lg *Log) setLastIndex(i int64) {
	lg.Lock()
	lg.lastIndex = i
	lg.Unlock()
}

func (lg *Log) GetLastIndex() int64 {
	lg.Lock()
	i := lg.lastIndex
	lg.Unlock()
	return i
}

func encode(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	le := LogEntry{Data: data}
	err := enc.Encode(le)
	return buf.Bytes(), err
}

func decode(dbytes []byte) (interface{}, error) {
	buf := bytes.NewBuffer(dbytes)
	enc := gob.NewDecoder(buf)
	var le LogEntry
	err := enc.Decode(&le)
	return le.Data, err
}

// int64 -> []byte
func toBytes(i int64) []byte {
	return []byte(strconv.FormatInt(i, 10)) // base 36
}

// []byte -> int64
func toIndex(key []byte) int64 {
	i, err := strconv.ParseInt(string(key), 10, 64) // base 36
	if err != nil {
		panic(err)
	}
	return i
}
