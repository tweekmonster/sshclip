package sshclip

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

var KeySize = 2049
var errKeyExists = errors.New("key exists")

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

func isAuthorizedKey(key ssh.PublicKey) bool {
	file, err := OpenDataFile("keys/authorized", os.O_RDONLY, 0)
	if err != nil {
		return false
	}

	keyBytes := key.Marshal()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fKey, _, _, _, err := ssh.ParseAuthorizedKey(scanner.Bytes())
		if err != nil {
			Elog(err)
			continue
		}

		if bytes.Equal(keyBytes, fKey.Marshal()) {
			return true
		}
	}

	return false
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

	file, err := OpenDataFile("keys/authorized", os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	authorizedKey := ssh.MarshalAuthorizedKey(key)
	_, err = file.Write(authorizedKey)
	return err
}

// FingerPrint returns the ssh.PublicKey SHA256 finger print.
func FingerPrint(key ssh.PublicKey) string {
	keyBytes := key.Marshal()
	h := sha256.New()
	h.Write(keyBytes)
	return fmt.Sprintf("SHA256:%s", base64.StdEncoding.EncodeToString(h.Sum(nil)))
	// return h.Sum(nil)
}

// // FingerPrintString returns the printable finger print.
// func FingerPrintString(s ssh.Signer) string {
// 	return base64.StdEncoding.EncodeToString(FingerPrint(s))
// 	// fp := ""
// 	//
// 	// for _, c := range FingerPrint(s) {
// 	// 	fp += fmt.Sprintf("%02x:", c)
// 	// }
// 	//
// 	// return fp[:len(fp)-1]
// }
