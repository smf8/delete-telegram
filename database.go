package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/gotd/td/session"
	"strings"
)

var ErrRecordNotFound = errors.New("record not found")

type Store struct {
	db *badger.DB
}

type badgerSessionSaver struct {
	userKey string
	db      *badger.DB
}

func initDB(c *Config) (*badger.DB, error) {
	dbOptions := badger.DefaultOptions(c.DatabasePath)
	dbOptions.EncryptionKey = []byte(c.DatabaseEncryptionKey)
	dbOptions.IndexCacheSize = 100 << 20

	db, err := badger.Open(dbOptions)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (b *badgerSessionSaver) StoreSession(ctx context.Context, data []byte) error {
	err := b.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(b.userSessionKey(b.userKey), data)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (b *badgerSessionSaver) LoadSession(ctx context.Context) ([]byte, error) {
	var sessionData []byte

	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(b.userSessionKey(b.userKey))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return session.ErrNotFound
			}

			return err
		}

		err = item.Value(func(val []byte) error {
			sessionData = append([]byte{}, val...)

			return nil
		})
		if err != nil {
			return err
		}

		if len(sessionData) == 0 {
			return fmt.Errorf("empty session found")
		}

		return nil
	})

	return sessionData, err
}

func (b *badgerSessionSaver) userSessionKey(key string) []byte {
	return []byte(key + "_session")
}

func (s *Store) StoreUserPassword(key, password string) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(s.userPasswordKey(key), []byte(password))
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (s *Store) GetUserPassword(key string) (string, error) {
	password := ""
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(s.userPasswordKey(key))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrRecordNotFound
			}

			return err
		}

		err = item.Value(func(val []byte) error {
			password = strings.Clone(string(val))

			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})

	return password, err
}

func (s *Store) userPasswordKey(key string) []byte {
	return []byte(key + "_password")
}
