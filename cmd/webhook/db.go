package main

import (
	"encoding/json"
	"errors"
	"slices"

	"github.com/tidwall/buntdb"
)

const usersKey = "users"

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

// readUsers returns the stored user list, treating a missing key as empty.
func readUsers(tx *buntdb.Tx) ([]string, error) {
	value, err := tx.Get(usersKey)
	if err != nil {
		if errors.Is(err, buntdb.ErrNotFound) {
			return []string{}, nil
		}
		return nil, err
	}
	var users []string
	if err := json.Unmarshal([]byte(value), &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (db *DB) getUsers() ([]string, error) {
	var users []string
	err := db.Db.View(func(tx *buntdb.Tx) error {
		var err error
		users, err = readUsers(tx)
		return err
	})
	return users, err
}

func (db *DB) CheckUser(key string) bool {
	users, err := db.getUsers()
	if err != nil {
		return false
	}
	return slices.Contains(users, key)
}

func (db *DB) CreateUser(token string) error {
	err := db.Db.Update(func(tx *buntdb.Tx) error {
		users, err := readUsers(tx)
		if err != nil {
			return err
		}
		value, err := json.Marshal(append(users, token))
		if err != nil {
			return err
		}
		_, _, err = tx.Set(usersKey, string(value), nil)
		return err
	})
	return err
}

func (db *DB) DeleteUser(token string) error {
	err := db.Db.Update(func(tx *buntdb.Tx) error {
		users, err := readUsers(tx)
		if err != nil {
			return err
		}
		newUsers := make([]string, 0, len(users))
		for _, user := range users {
			if user != token {
				newUsers = append(newUsers, user)
			}
		}
		value, err := json.Marshal(&newUsers)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(usersKey, string(value), nil)
		return err
	})
	return err
}

func (db *DB) InitDB() error {
	instance, err := buntdb.Open(db.path)
	if err != nil {
		return err
	}
	db.Db = instance
	return nil
}
