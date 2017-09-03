package yhdbt

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"strconv"
)

type DBOpt struct {
	DBConn *leveldb.DB
}

var GDBOpt = &DBOpt{}

func (this *DBOpt) Open(dbPath string) error {
	var err error
	if this.DBConn, err = leveldb.OpenFile(dbPath, nil); err != nil {
		return fmt.Errorf(`[DBOPT] open leveldb error:`, err)
	}
	return nil
}

func (this *DBOpt) Close() {
	if this.DBConn != nil {
		this.DBConn.Close()
	}
	this.DBConn = nil
}

func (this *DBOpt) GetValue(key []byte) []byte {
	if data, err := this.DBConn.Get([]byte(key), nil); err != nil {
		log.Println(`[BDOPT] Get error:`, err)
		return []byte{}
	} else {
		return data
	}
}

func (this *DBOpt) GetValueAsInt(key []byte) int {
	val := this.GetValue(key)
	if len(val) == 0 {
		return 0
	}
	n, err := strconv.Atoi(string(val[:]))
	if err != nil {
		log.Println(`[BDOPT] Atoi error:`, err)
		return 0
	}
	return n
}

func (this *DBOpt) PutValue(key []byte, val []byte) error {
	return this.DBConn.Put(key, val, nil)
}

func (this *DBOpt) PutValueInt(key []byte, val int) error {
	return this.DBConn.Put(key, []byte(fmt.Sprint(val)), nil)
}

func (this *DBOpt) DelKey(key []byte) error {
	return this.DBConn.Delete(key, nil)
}

func (this *DBOpt) PutBatch(m map[string][]byte) error {
	batch := new(leveldb.Batch)
	for k, v := range m {
		batch.Put([]byte(k), v)
	}
	return this.DBConn.Write(batch, nil)
}

func (this *DBOpt) GetBatch(m map[string][]byte) {
	for k, _ := range m {
		m[k] = this.GetValue([]byte(k))
	}
}
