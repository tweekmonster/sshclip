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
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
)

var KeySize = 2049
var errKeyExists = errors.New("key exists")
var errKeyNoRecord = errors.New("key has no record")

type KeyRecord struct {
	Added time.Time `json:"added"`
	IP    string    `json:"ip"`
	State string    `json:"state"`
}

type PublicKeyRecord struct {
	ssh.PublicKey
	KeyRecord
}

type KeyReviewItem struct {
	FingerPrint []byte
	KeyRecord
}

func NewPublicKeyRecord(key ssh.PublicKey, addr net.Addr) PublicKeyRecord {
	rec := PublicKeyRecord{
		PublicKey: key,
		KeyRecord: KeyRecord{
			Added: time.Now(),
			State: "pending",
		},
	}

	if addr != nil {
		ip, _, err := net.SplitHostPort(addr.String())
		if err != nil {
			ip = addr.String()
		}
		rec.IP = ip
	}

	return rec
}

func (p PublicKeyRecord) MarshalComment() []byte {
	data, _ := json.Marshal(p.KeyRecord)
	return data
}

func GetPublicKeyRecord(key ssh.PublicKey) (KeyRecord, error) {
	switch v := key.(type) {
	case PublicKeyRecord:
		return v.KeyRecord, nil
	}
	return KeyRecord{}, errKeyNoRecord
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

	file, err := OpenDataFile(filepath.Join("keys", filename), os.O_RDONLY, 0)
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

func writeKeys(filename string, keys []ssh.PublicKey, eval ssh.PublicKey, add bool) error {
	file, err := OpenDataFile(filepath.Join("keys", filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	defer file.Close()

	file.Write([]byte("## DO NOT EDIT!\n"))

	if eval != nil {
		evalBytes := eval.Marshal()
		ikeys := keys
		keys = keys[0:0]

		for _, key := range ikeys {
			if bytes.Equal(evalBytes, key.Marshal()) {
				continue
			}
			keys = append(keys, key)
		}

		if add {
			keys = append(keys, eval)
		}
	}

	for _, key := range keys {
		keyBytes := ssh.MarshalAuthorizedKey(key)

		switch k := key.(type) {
		case PublicKeyRecord:
			keyBytes = append(bytes.TrimSpace(keyBytes), ' ')
			keyBytes = append(keyBytes, k.MarshalComment()...)
			keyBytes = append(keyBytes, '\n')
		}

		if _, err := file.Write(keyBytes); err != nil {
			return err
		}
	}

	return nil
}

func KeyExists(filename string, key ssh.PublicKey) bool {
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

func AddKey(filename string, key ssh.PublicKey) error {
	keys, err := readKeys(filename)
	if err != nil {
		return err
	}

	var out []ssh.PublicKey
	keyBytes := key.Marshal()

	for _, k := range keys {
		if !bytes.Equal(keyBytes, k.Marshal()) {
			out = append(out, k)
		}
	}

	out = append(out, key)
	return writeKeys(filename, out, key, true)
}

func RemoveKey(filename string, key ssh.PublicKey) error {
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

	return writeKeys(filename, outKeys, key, false)
}

func allKeys() (keys []PublicKeyRecord) {
	for _, loc := range []string{"pending", "authorized", "rejected"} {
		keySet, err := readKeys(loc)
		if err != nil {
			continue
		}

		for _, k := range keySet {
			switch v := k.(type) {
			case PublicKeyRecord:
				keys = append(keys, v)
			default:
				rec := NewPublicKeyRecord(k, nil)
				rec.State = loc
				keys = append(keys, rec)
			}
		}
	}

	return
}

func findFingerPrint(filename string, fingerprint []byte) (PublicKeyRecord, error) {
	keys, err := readKeys(filename)
	if err != nil {
		return PublicKeyRecord{}, err
	}

	for _, k := range keys {
		if bytes.Equal(fingerprint, FingerPrintBytes(k)) {
			switch v := k.(type) {
			case PublicKeyRecord:
				return v, nil
			default:
				rec := NewPublicKeyRecord(k, nil)
				rec.State = filename
				return rec, nil
			}
		}
	}

	return PublicKeyRecord{}, ErrNotExist
}

func FindFingerPrint(fingerprint []byte) (rec PublicKeyRecord, err error) {
	for _, loc := range []string{"pending", "authorized", "rejected"} {
		rec, err = findFingerPrint(loc, fingerprint)
		if err == nil {
			return
		}
	}

	err = ErrNotExist
	return
}

// IsAuthorizedKey checks if a key is authorized.  If the authorized keys file
// does not exist, save the key and return true.
func IsAuthorizedKey(key ssh.PublicKey) bool {
	if !DataFileExists("keys/authorized") {
		if err := AddKey("authorized", key); err == nil {
			return true
		}
		return false
	}

	return KeyExists("authorized", key)
}

func FingerPrintBytes(key ssh.PublicKey) []byte {
	keyBytes := key.Marshal()
	h := sha256.New()
	h.Write(keyBytes)
	return h.Sum(nil)
}

// FingerPrint returns the ssh.PublicKey SHA256 finger print.
func FingerPrint(key ssh.PublicKey) string {
	return fmt.Sprintf("SHA256:%s", base64.StdEncoding.EncodeToString(FingerPrintBytes(key)))
}
