package sshclip

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

var KeySize = 2049
var errKeyExists = errors.New("key exists")

type KeyRecord struct {
	DateAdded time.Time `json:"added"`
	IP        string    `json:"ip"`
}

type PublicKeyRecord struct {
	ssh.PublicKey
	KeyRecord
}

func NewPublicKeyRecord(key ssh.PublicKey, addr net.Addr) PublicKeyRecord {
	ip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		ip = addr.String()
	}

	return PublicKeyRecord{
		PublicKey: key,
		KeyRecord: KeyRecord{
			DateAdded: time.Now(),
			IP:        ip,
		},
	}
}

func (p PublicKeyRecord) MarshalComment() []byte {
	data, _ := json.Marshal(p.KeyRecord)
	return data
}

// Generates an SSH public and private key then writes them to a file.
func generateKey(path string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, KeySize)
	if err != nil {
		return err
	}

	privateKeyDer := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privateKeyDer,
	})

	sshPublicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	authorizedKey := ssh.MarshalAuthorizedKey(sshPublicKey)
	if err := WriteData(path+".pub", authorizedKey, 0600); err != nil {
		return err
	}

	if err := WriteData(path, privateKeyPem, 0600); err != nil {
		return err
	}

	return nil
}

func loadKey(path string) (ssh.Signer, error) {
	if !DataFileExists(path) {
		generateKey(path)
	}

	data, err := ReadData(path)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(data)
}

// GetHostKey returns the host's private key signer.  It's generated if it
// doesn't exist.
func GetHostKey() (ssh.Signer, error) {
	return loadKey("keys/server")
}

// GetClientKey returns the clients's private key signer.  It's generated if it
// doesn't exist.
func GetClientKey() (ssh.Signer, error) {
	return loadKey("keys/client")
}

func readKeys(filename string) ([]ssh.PublicKey, error) {
	var keys []ssh.PublicKey

	file, err := OpenDataFile(filename, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return keys, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		keyBytes := scanner.Bytes()
		if keyBytes[0] == '#' {
			continue
		}

		fKey, comment, _, _, err := ssh.ParseAuthorizedKey(keyBytes)
		if err != nil {
			Elog(err, string(keyBytes))
			continue
		}

		if comment != "" {
			keyRec := PublicKeyRecord{
				PublicKey: fKey,
			}

			if err := json.Unmarshal([]byte(comment), &keyRec.KeyRecord); err == nil {
				Dlog("Loaded record: %#v", keyRec)
				keys = append(keys, keyRec)
				continue
			} else {
				Elog(err)
			}
		}

		keys = append(keys, fKey)
	}

	return keys, nil
}

func writeKeys(filename string, keys []ssh.PublicKey) error {
	file, err := OpenDataFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	defer file.Close()

	file.Write([]byte("## DO NOT EDIT!\n"))

	for _, key := range keys {
		keyBytes := ssh.MarshalAuthorizedKey(key)

		switch k := key.(type) {
		case PublicKeyRecord:
			keyBytes = append(bytes.TrimSpace(keyBytes), ' ')
			keyBytes = append(keyBytes, k.MarshalComment()...)
		}

		if _, err := file.Write(keyBytes); err != nil {
			return err
		}
	}

	return nil
}

func keyExists(filename string, key ssh.PublicKey) bool {
	keys, err := readKeys(filename)
	if err != nil {
		return false
	}

	keyBytes := key.Marshal()
	for _, key := range keys {
		if bytes.Equal(keyBytes, key.Marshal()) {
			return true
		}
	}

	return false
}

func addKey(filename string, key ssh.PublicKey) error {
	keys, err := readKeys(filename)
	if err != nil {
		return err
	}

	keys = append(keys, key)
	return writeKeys(filename, keys)
}

func removeKey(filename string, key ssh.PublicKey) error {
	keys, err := readKeys(filename)
	if err != nil {
		return err
	}

	var outKeys []ssh.PublicKey
	keyBytes := key.Marshal()
	for _, key := range keys {
		if !bytes.Equal(keyBytes, key.Marshal()) {
			outKeys = append(outKeys, key)
		}
	}

	return writeKeys(filename, outKeys)
}

func isAuthorizedKey(key ssh.PublicKey) bool {
	return keyExists("keys/authorized", key)
}

// IsAuthorizedKey checks if a key is authorized.  If the authorized keys file
// does not exist, save the key and return true.
func IsAuthorizedKey(key ssh.PublicKey) bool {
	if !DataFileExists("keys/authorized") {
		if err := AddAuthorizedKey(key); err == nil {
			return true
		}
		return false
	}

	return isAuthorizedKey(key)
}

// AddAuthorizedKey adds a key to the authorized keys file.
func AddAuthorizedKey(key ssh.PublicKey) error {
	if isAuthorizedKey(key) {
		return errKeyExists
	}

	return addKey("keys/authorized", key)
}

// FingerPrint returns the ssh.PublicKey SHA256 finger print.
func FingerPrint(key ssh.PublicKey) string {
	keyBytes := key.Marshal()
	h := sha256.New()
	h.Write(keyBytes)
	return fmt.Sprintf("SHA256:%s", base64.StdEncoding.EncodeToString(h.Sum(nil)))
}

func IsPendingKey(key ssh.PublicKey) bool {
	return keyExists("keys/pending", key)
}

func AddPendingKey(key ssh.PublicKey) error {
	return addKey("keys/pending", key)
}

func RemovePendingKey(key ssh.PublicKey) error {
	return removeKey("keys/pending", key)
}
