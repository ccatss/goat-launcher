package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"os"

	"golang.org/x/crypto/argon2"
)

type PromptFunc func(string) (string, error)

func NewFileStore(path, passphrase string) Store {
	return &fileStore{
		file: path,
	}
}

type fileStore struct {
	file       string
	passphrase string
}

func (d *fileStore) loadData() (map[string]Session, error) {
	fileData, err := os.ReadFile(d.file)

	if err != nil {
		return nil, err
	}

	var m map[string]Session

	if d.passphrase == "" {
		if err := json.Unmarshal(fileData, &m); err != nil {
			return nil, err
		}
	} else {
		if err := DecryptJSONWithPassphrase(fileData, &m, d.passphrase); err != nil {
			return nil, err
		}
	}

	return m, nil
}

func (d *fileStore) saveData(m map[string]Session) error {
	data, err := json.Marshal(m)

	if err != nil {
		return err
	}

	if d.passphrase != "" {
		data, err = EncryptJSONWithPassphrase(data, d.passphrase)

		if err != nil {
			return err
		}
	}

	return os.WriteFile(d.file, data, 0600)
}

func (d *fileStore) Mutate(f func(m map[string]Session)) error {
	data, err := d.loadData()

	if err != nil {
		return err
	}

	f(data)

	return d.saveData(data)
}

func (d *fileStore) Set(name string, session Session) error {
	return d.Mutate(func(m map[string]Session) {
		m[name] = session
	})
}

func (d *fileStore) Get(name string) (*Session, error) {
	data, err := d.loadData()

	if err != nil {
		return nil, err
	}

	account, ok := data[name]

	if !ok {
		return nil, errors.New("account not found")
	}

	return &account, nil
}

func (d *fileStore) Delete(name string) error {
	return d.Mutate(func(m map[string]Session) {
		delete(m, name)
	})
}

func (d *fileStore) List() ([]string, error) {
	data, err := d.loadData()

	if err != nil {
		return nil, err
	}

	var keys []string

	for k := range data {
		keys = append(keys, k)
	}

	return keys, nil
}

const (
	argon2Time    = 1         // Number of passes over the memory
	argon2Memory  = 64 * 1024 // 64 MB of memory usage
	argon2Threads = 4         // Number of parallel threads to use
	keyLength     = 32        // We need exactly 32 bytes for AES-256
	saltLength    = 16        // 16 bytes is the standard cryptographic salt size
)

// deriveKey mixes a text passphrase and a salt using Argon2id to create a 32-byte key
func deriveKey(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, argon2Time, argon2Memory, argon2Threads, keyLength)
}

// EncryptJSONWithPassphrase derives a key from a passphrase, encrypts the JSON, and prepends the salt to the file.
func EncryptJSONWithPassphrase(data []byte, passphrase string) ([]byte, error) {
	salt := make([]byte, saltLength)

	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	secretKey := deriveKey(passphrase, salt)

	block, err := aes.NewCipher(secretKey)

	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// 5. Build the final file layout: [ 16 bytes Salt ] + [ 12 bytes Nonce ] + [ Ciphertext ]
	finalPayload := make([]byte, 0, len(salt)+len(nonce)+len(ciphertext))
	finalPayload = append(finalPayload, salt...)
	finalPayload = append(finalPayload, nonce...)
	finalPayload = append(finalPayload, ciphertext...)

	return finalPayload, nil
}

// DecryptJSONWithPassphrase extracts the salt from the file header, recreates the key, and decrypts the file.
func DecryptJSONWithPassphrase(fileData []byte, target interface{}, passphrase string) error {
	block, err := aes.NewCipher(make([]byte, keyLength))

	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		return err
	}

	nonceSize := gcm.NonceSize()

	minExpectedSize := saltLength + nonceSize

	if len(fileData) < minExpectedSize {
		return errors.New("encrypted file payload is corrupted or too short")
	}

	salt := fileData[:saltLength]
	nonce := fileData[saltLength : saltLength+nonceSize]
	pureCiphertext := fileData[saltLength+nonceSize:]

	secretKey := deriveKey(passphrase, salt)

	block, err = aes.NewCipher(secretKey)

	if err != nil {
		return err
	}

	gcm, err = cipher.NewGCM(block)

	if err != nil {
		return err
	}

	plaintext, err := gcm.Open(nil, nonce, pureCiphertext, nil)

	if err != nil {
		return errors.New("decryption failed: invalid passphrase or modified data")
	}

	return json.Unmarshal(plaintext, target)
}
