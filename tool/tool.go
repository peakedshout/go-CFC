package tool

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	"io"
	"log"
	"net"
	"strings"
)

func NewId(n int) (str string) {
	for i := n; i > 0; i-- {
		uid, err := uuid.NewV4()
		if err != nil {
			panic(err)
		}
		uidStr := uid.String()
		s := strings.Replace(uidStr, "-", "", -1)
		str += s
	}
	return
}

func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func MustUnmarshal(b []byte, body interface{}) {
	if len(b) == 0 {
		return
	}
	err := json.Unmarshal(b, &body)
	if err != nil {
		fmt.Println(string(b), "err:", err)
		panic(err)
	}
}
func MustMarshal(body interface{}) []byte {
	b, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return b
}

func UnmarshalV2(inBody interface{}, outBody interface{}) error {
	b, err := json.Marshal(inBody)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, outBody)
}
func MustUnmarshalV2(inBody interface{}, outBody interface{}) {
	MustUnmarshal(MustMarshal(inBody), &outBody)
}

func Println(conn net.Conn, i ...any) {
	var iList []any
	iList = append(iList, conn.LocalAddr(), "->", conn.RemoteAddr())
	for _, one := range i {
		iList = append(iList, one)
	}
	log.Println(iList...)
}

func toolHash(b []byte) []byte {
	s := sha256.New()
	s.Write(b)
	return s.Sum(nil)
}

func checkHash(h []byte, data []byte) bool {
	h2 := toolHash(data)
	if len(h) != len(h2) {
		return false
	}
	for i := range h2 {
		if h[i] != h2[i] {
			return false
		}
	}
	return true
}

const mashBytesTag = "go-CFC-Data"

type MashBytes struct {
	Data string `json:"go-CFC-Data"`
}

func MustBytesToBase64(b []byte) MashBytes {
	return MashBytes{Data: base64.StdEncoding.EncodeToString(b)}
}
func MustBase64ToBytes(str string) []byte {
	b, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return b
}

func MustResolveTCPAddr(addr net.Addr) *net.TCPAddr {
	tAddr, _ := net.ResolveTCPAddr(addr.Network(), addr.String())
	return tAddr
}
