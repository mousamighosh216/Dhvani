// Package auth: PBKDF2 password hashing and HS256 JWTs, standard library only.
package auth

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const iterations = 210_000

var enc = base64.RawURLEncoding

func HashPassword(pw string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, pw, salt, iterations, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2$%d$%s$%s", iterations, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

func CheckPassword(pw, stored string) bool {
	p := strings.Split(stored, "$")
	if len(p) != 4 || p[0] != "pbkdf2" {
		return false
	}
	n, err := strconv.Atoi(p[1])
	if err != nil || n < 1 {
		return false
	}
	salt, err1 := enc.DecodeString(p[2])
	want, err2 := enc.DecodeString(p[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, pw, salt, n, len(want))
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

type claims struct {
	Sub int   `json:"sub"`
	Exp int64 `json:"exp"`
}

func sign(secret []byte, signing string) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(signing))
	return enc.EncodeToString(m.Sum(nil))
}

func Sign(secret []byte, userID int, ttl time.Duration) string {
	body, _ := json.Marshal(claims{userID, time.Now().Add(ttl).Unix()})
	head := enc.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	signing := head + "." + enc.EncodeToString(body)
	return signing + "." + sign(secret, signing)
}

var ErrInvalidToken = errors.New("invalid or expired token")

func Verify(secret []byte, token string) (int, error) {
	p := strings.Split(token, ".")
	if len(p) != 3 {
		return 0, ErrInvalidToken
	}
	if !hmac.Equal([]byte(sign(secret, p[0]+"."+p[1])), []byte(p[2])) {
		return 0, ErrInvalidToken
	}
	body, err := enc.DecodeString(p[1])
	if err != nil {
		return 0, ErrInvalidToken
	}
	var c claims
	if json.Unmarshal(body, &c) != nil || c.Sub <= 0 || time.Now().Unix() >= c.Exp {
		return 0, ErrInvalidToken
	}
	return c.Sub, nil
}
