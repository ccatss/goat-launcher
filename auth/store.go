package auth

import (
	"github.com/99designs/keyring"
)

const serviceName = "goat"

type Store interface {
	Set(name string, session Session) error
	Get(name string) (*Session, error)
	Delete(name string) error
	List() ([]string, error)
}

func NewStore() (Store, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName:  serviceName,
		KeychainName: "accounts",
	})

	if err != nil {
		return nil, err
	}

	return &keyringStore{
		ring: ring,
	}, nil
}
