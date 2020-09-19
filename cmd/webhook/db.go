package main

import (
	"encoding/json"
	"github.com/tidwall/buntdb"
)

type DB struct {
	path string
	Db   *buntdb.DB
}

func NewDB(path string) (*DB, error) {
	db := &DB{
		path: path,
		Db:   nil,
	}
	err := db.InitDB()
	if err != nil {
		return nil, err
	}
	var config buntdb.Config
	if err := db.Db.ReadConfig(&config); err != nil {
		return nil, err
	}
	config.SyncPolicy = buntdb.Always
	if err := db.Db.SetConfig(config); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) setKey(key string, value string) error {
	err := db.Db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, value, nil)
		return err
	})
	return err
}

func (db *DB) getKey(key string) (string, error) {
	var value string
	var err error
	err = db.Db.View(func(tx *buntdb.Tx) error {
		dbValue, err := tx.Get(key)
		if err != nil {
			return err
		}
		value = dbValue
		return nil
	})
	return value, err
}

func (db *DB) getUsers() []string {
	dataString, err := db.getKey("users")
	if err != nil {
		return []string{}
	}
	var data []string
	err = json.Unmarshal([]byte(dataString), &data)
	if err != nil {
		return []string{}
	}
	return data
}

func (db *DB) CheckUser(key string) bool {
	users := db.getUsers()
	for _, user := range users {
		if user == key {
			return true
		}
	}
	return false
}

func (db *DB) CreateUser(token string) error {
	tokens := append(db.getUsers(), token)
	value, err := json.Marshal(&tokens)
	if err != nil {
		return err
	}
	return db.setKey("users", string(value))
}

func (db *DB) InitDB() error {
	instance, err := buntdb.Open(db.path)
	if err != nil {
		return err
	}
	db.Db = instance
	return nil
}

func (db *DB) DeleteUser(token string) error {
	tokens := db.getUsers()
	newTokens := make([]string, 0)
	for index := range tokens {
		if tokens[index] != token {
			newTokens = append(newTokens, tokens[index])
		}
	}
	value, err := json.Marshal(&newTokens)
	if err != nil {
		return err
	}
	return db.setKey("users", string(value))
}
