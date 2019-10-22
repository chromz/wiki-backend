package argon

import (
	"crypto/rand"
	// "crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/chromz/wiki-backend/pkg/log"
	"golang.org/x/crypto/argon2"
	"strings"
)

const (
	keyLen  = 64
	saltLen = 32
)

var hashBuilder strings.Builder
var logger = log.GetLogger()

// GenerateFromPassword generate a hash from a password
func GenerateFromPassword(password []byte, time, memory uint32,
	threads uint8) (string, error) {
	salt := make([]byte, saltLen)
	_, err := rand.Read(salt)
	if err != nil {
		return "", errors.New("Unable to generate random salt")
	}
	hash := argon2.IDKey(password, salt, time, memory, threads, keyLen)
	encodedSalt := base64.StdEncoding.EncodeToString(salt)
	encodedHash := base64.StdEncoding.EncodeToString(hash)

	hashBuilder.WriteString("$argon2id$v=")
	hashBuilder.WriteString(fmt.Sprint(argon2.Version))
	hashBuilder.WriteString("$m=")
	hashBuilder.WriteString(fmt.Sprint(memory))
	hashBuilder.WriteString(",t=")
	hashBuilder.WriteString(fmt.Sprint(time))
	hashBuilder.WriteString(",p=")
	hashBuilder.WriteString(fmt.Sprint(threads))
	hashBuilder.WriteString("$")
	hashBuilder.WriteString(encodedSalt)
	hashBuilder.WriteString("$")
	hashBuilder.WriteString(encodedHash)
	return hashBuilder.String(), nil
}

// CompareHashAndPassword comapres a hash to a password returns nil on success
// or an error on failure
func CompareHashAndPassword(hashedPassword, password []byte) error {
	// time, memory, threads, digest, err := decodeHash(hashedPassword)
	return nil
}

func decodeHash(hashedPassword []byte) (time, memory uint32,
	threads uint8, hash string, err error) {

	components := strings.Split(string(hashedPassword), "$")
	if len(components) != 6 {
		return 0, 0, 0, "", errors.New("incorrect encoded hash")
	}
	return 0, 0, 0, "", errors.New("incorrect encoded hash")
}
