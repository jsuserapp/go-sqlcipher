package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"time"
)

/*两种方式设置的密码效果是一样的，但是无密码数据库和有秘密数据库不能互相切换，一旦创建就不能改变*/

import (
	_ "github.com/jsuserapp/go-sqlcipher"
)

const (
	keydb    = "./data/key.db"
	phrasedb = "./data/phrase.db"
	normaldb = "./data/normal.db"
)

const (
	createSqliteLogTab = `CREATE TABLE IF NOT EXISTS log (
    id INTEGER PRIMARY KEY, -- 在 SQLite 中, INTEGER PRIMARY KEY 默认就是自增的
    tag TEXT NOT NULL DEFAULT '',
    log TEXT NOT NULL,
    trace TEXT NOT NULL,
    color TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP -- SQLite 不支持 DATETIME 的精度定义
);`
	createSqliteLogTabIdx = `CREATE INDEX IF NOT EXISTS idx_tag_created_at ON log (tag, created_at);`
)
const (
	key1    = "6635a52a804fc16200f9f29bb34da84cd8ee9191782eee6bab544546999f2da7"
	key2    = "46961876fe36415538749f8d09ea3a9dfa58c282cfbd3eee954a85474326d7c8"
	phrase1 = "helloworld"
	phrase2 = "HELLOWORLD"
)

func main() {
	err := os.MkdirAll("./data", os.ModePerm)
	if err != nil {
		panic(err)
	}
	testNormalDb()
	//注意：第二次运行，因为密码已经被修改，会打开失败，必须更换为新的密码
	testKeyDb()
	testPhraseDb()
	testChangePhrase()
	testChangeKey()
}

func createLogTable(db *sql.DB) {
	_, err := db.Exec(createSqliteLogTab)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(createSqliteLogTabIdx)
	if err != nil {
		panic(err)
	}
}
func insertLog(db *sql.DB, log string) int64 {
	sqlCase := "INSERT INTO log (color,trace,log,created_at,tag) VALUES (?,?,?,?,?)"
	createdAt := time.Now().Format("2006-01-02 15:04:05.000")
	rst, err := db.Exec(sqlCase, "red", "sqlcipher.go:34", log, createdAt, "")
	if err != nil {
		panic(err)
	}
	id, _ := rst.LastInsertId()
	return id
}
func readLog(db *sql.DB) {
	sqlCase := "SELECT color,trace,log,created_at FROM log WHERE tag=?"
	rows, err := db.Query(sqlCase, "")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var color, trace, log string
		var createdAt time.Time
		err = rows.Scan(&color, &trace, &log, &createdAt)
		if err != nil {
			fmt.Println(err)
		} else {
			ts := createdAt.Format("2006-01-02 15:04:05.000")
			fmt.Println(color, ts, trace, log)
		}
	}
}

func openNormalDb() *sql.DB {
	db, err := sql.Open("sqlite3", normaldb)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return db
}
func openKeyDb(key32 []byte) *sql.DB {
	if len(key32) != 32 {
		fmt.Println("密钥长度必须32字节")
		return nil
	}
	key := hex.EncodeToString(key32)
	dataSourceName := fmt.Sprintf("%s?_pragma_key=x'%s'&_pragma_cipher_page_size=4096", keydb, key)
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return db
}
func openPhraseDb(phrase string) *sql.DB {
	key := url.QueryEscape(phrase)
	dataSourceName := fmt.Sprintf("%s?_pragma_key=%s&_pragma_cipher_page_size=4096", phrasedb, key)
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return db
}

func changeDbKey(db *sql.DB, key32 []byte) {
	if len(key32) != 32 {
		fmt.Println("密钥长度必须32字节")
		return
	}
	key := hex.EncodeToString(key32)

	_, err := db.Exec(fmt.Sprintf("PRAGMA rekey = \"x'%s'\"", key))
	if err != nil {
		fmt.Println(err)
	}
}

func changeDbPhrase(db *sql.DB, phrase string) {
	//不能使用？模式来设置密码，只能使用字符串模式替换
	phrase = url.QueryEscape(phrase)
	_, err := db.Exec(fmt.Sprintf("PRAGMA rekey = %s", phrase))
	if err != nil {
		fmt.Println(err)
	}
}
func testKeyDb() {
	key32, err := hex.DecodeString(key1)
	if err != nil {
		panic(err)
	}
	db := openKeyDb(key32)
	if db == nil {
		return
	}
	readWriteDb(db)
}
func testPhraseDb() {
	db := openPhraseDb(phrase1)
	if db == nil {
		return
	}
	readWriteDb(db)
}
func testNormalDb() {
	db := openNormalDb()
	if db == nil {
		return
	}
	readWriteDb(db)
}
func testChangePhrase() {
	db := openPhraseDb(phrase1)
	if db == nil {
		return
	}
	//change to new password
	//This will decrypt all data and re-encrypt it, so it is a time-consuming operation.
	changeDbPhrase(db, phrase2)
	readWriteDb(db)
}
func testChangeKey() {
	key32, err := hex.DecodeString(key1)
	if err != nil {
		panic(err)
	}
	db := openKeyDb(key32)
	if db == nil {
		return
	}
	newKey32, err := hex.DecodeString(key2)
	changeDbKey(db, newKey32)
	readWriteDb(db)
}

// This will actually create the database
func readWriteDb(db *sql.DB) {
	if db == nil {
		return
	}
	defer func() {
		_ = db.Close()
	}()

	var version string
	err := db.QueryRow(`SELECT sqlite_version()`).Scan(&version)
	if err != nil {
		println(err)
		return
	}

	fmt.Printf("SQLite Version: %s\n", version)
	createLogTable(db)
	insertLog(db, "test log")
	readLog(db)
}
