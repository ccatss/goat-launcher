package auth

import (
	"encoding/json"
	"errors"

	"github.com/99designs/keyring"
)

const serviceName = "goat"

type Store interface {
	Set(name string, session Session) error
	Get(name string) (*Session, error)
	Delete(name string) error
	List() ([]string, error)
}

type defaultStore struct {
	accounts map[string]Session
}

func (d *defaultStore) Set(name string, session Session) error {
	d.accounts[name] = session

	return nil
}

func (d *defaultStore) Get(name string) (*Session, error) {
	account, ok := d.accounts[name]

	if !ok {
		return nil, errors.New("account not found")
	}

	return &account, nil
}

func (d *defaultStore) Delete(name string) error {
	delete(d.accounts, name)
	return nil
}

func (d *defaultStore) List() ([]string, error) {
	var keys []string

	for k := range d.accounts {
		keys = append(keys, k)
	}

	return keys, nil
}

func NewStore() Store {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: serviceName,
	})

	if err != nil {
		return &defaultStore{}
	}

	return &keyringStore{
		ring: ring,
	}
}

type keyringStore struct {
	ring keyring.Keyring
}

func (k *keyringStore) Set(name string, session Session) error {
	data, err := json.Marshal(session)

	if err != nil {
		return err
	}

	return k.ring.Set(keyring.Item{
		Key:  name,
		Data: data,
	})
}

func (k *keyringStore) Get(name string) (*Session, error) {
	val, err := k.ring.Get(name)

	if err != nil {
		return nil, err
	}

	var session Session

	if err := json.Unmarshal(val.Data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (k *keyringStore) Delete(name string) error {
	return k.ring.Remove(name)
}

func (k *keyringStore) List() ([]string, error) {
	return k.ring.Keys()
}
