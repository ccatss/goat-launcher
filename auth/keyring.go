package auth

import (
	"encoding/json"

	"github.com/99designs/keyring"
)

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
