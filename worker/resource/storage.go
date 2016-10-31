package main

import (
	"encoding/json"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/jmoiron/sqlx"
)

func init() {
	initMysql()
}

//==== Memory Store
// var keyResultMap map[string]map[string]int

// func initMem() {
// 	keyResultMap = make(map[string]map[string]int)
// }

// func store(key string, result map[string]int) {
// 	keyResultMap[key] = result
// }

// func findKey(key string) map[string]int {
// 	if v, ok := keyResultMap[key]; ok {
// 		return v
// 	}
// 	return nil
// }

// ===== Mysql Store =====
var mysqlDB *sqlx.DB
var mysqlConn = ""

func initMysql() {
	if len(mysqlConn) > 0 {
		db, err := sqlx.Connect("mysql", mysqlConn)
		if err != nil {
			panic(err)
		}

		glog.Infof("connect to mysql success: %s", mysqlConn)
		mysqlDB = db
	} else {
		glog.Infof("mysqlConn is empty")
	}
}

func store(key string, result map[string]int) error {
	b, err := json.Marshal(result)
	if err != nil {
		glog.Error(err)
		return err
	}
	resultStr := string(b)
	sql := "INSERT INTO resources (`key`, `result`) VALUES (?, ?) ON DUPLICATE KEY UPDATE result = ?"

	_, err = mysqlDB.Exec(sql, key, resultStr, resultStr)
	if err != nil {
		glog.Error(err)
		return err
	}

	return nil
}

func findKey(key string) map[string]int {
	var result string
	var resultMap map[string]int
	sql := "select result from resources where `key` = ? limit 1"
	row := mysqlDB.QueryRow(sql, key)
	err := row.Scan(&result)
	if err != nil {
		glog.Error(err)
		return nil
	}

	err = json.Unmarshal([]byte(result), &resultMap)
	if err != nil {
		glog.Error(err)
		return nil
	}

	return resultMap
}
