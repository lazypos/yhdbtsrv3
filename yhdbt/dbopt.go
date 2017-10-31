package yhdbt

import (
	"bytes"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"log"
	"sort"
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

func (this *DBOpt) GetMaxScore(max int) ([]*RankScoreInfo, error) {
	type info struct {
		Score int
		Uid   string
		Nick  string
	}

	arr := make([]int, max)
	m := make([]*info, 5)
	count := 0
	lowLevel := -1 //积分下限

	iter := this.DBConn.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		idx := bytes.LastIndex(key, []byte("_score"))
		//log.Println(string(key[:]), string(iter.Value()[:]))
		if idx > 0 {
			n, err := strconv.Atoi(string(iter.Value()[:]))
			if err != nil {
				log.Println(err)
				continue
			}
			//log.Println(string(iter.Key()[:]), string(iter.Value()[:]))
			uid := string(key[:idx])
			//必须要玩过一局
			zong := this.GetValueAsInt([]byte(fmt.Sprintf(`%s_win`, uid)))
			if zong == 0 {
				continue
			}
			if count < 5 {
				arr[count] = n
				if lowLevel < 0 {
					lowLevel = n
				}
				if lowLevel > 0 && n < lowLevel {
					lowLevel = n
				}
				nick := string(this.GetValue([]byte(fmt.Sprintf(`%s_nick`, uid)))[:])
				m[count] = &info{Score: n, Uid: uid, Nick: nick}
				//log.Println(count, m, nick, uid, n)
				count++
			} else {
				if n >= lowLevel {
					sort.Ints(arr)
					low := arr[0]
					//log.Println(arr, low, n)
					if n > low {
						arr[0] = n
						sort.Ints(arr)
						lowLevel = arr[0]
						for i, v := range m {
							if v.Score == low {
								nick := string(this.GetValue([]byte(fmt.Sprintf(`%s_nick`, uid)))[:])
								m[i].Score = n
								m[i].Uid = uid
								m[i].Nick = nick
								break
							}
						}
					}
				}
			}
		}
	}
	iter.Release()
	sort.Ints(arr)
	//log.Println(arr, m)
	rst := make([]*RankScoreInfo, max)
	for i := 0; i < max; i++ {
		nowScore := arr[max-i-1]
		if nowScore == 0 {
			continue
		}
		//log.Println(i, nowScore)
		for _, v := range m {
			//log.Println(v.Nick, v.Score, v.Uid)
			if v.Score == nowScore {
				//log.Println(i, nowScore, v.Nick, v.Score, v.Uid)
				rst[i] = &RankScoreInfo{socre: fmt.Sprintf(`%d`, nowScore), nick: v.Nick}
				v.Score = -1
				break
			}
		}
	}
	return rst, iter.Error()
}

func (this *DBOpt) test() {
	buf := bytes.NewBufferString("")
	iter := this.DBConn.NewIterator(nil, nil)
	for iter.Next() {
		buf.Write(iter.Key())
		buf.WriteString("=")
		buf.Write(iter.Value())
		buf.WriteString("\r\n")
	}
	ioutil.WriteFile("f:/test/a.txt", buf.Bytes(), 0x666)
}

func ParseDB() {
	dbopt := &DBOpt{}
	dbopt.Open("f:/test/yhdbt_db")
	dbopt.test()
	defer dbopt.Close()
}
