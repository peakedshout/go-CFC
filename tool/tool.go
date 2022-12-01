package tool

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
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

func GetConnBs(reader *bufio.Reader) (b []byte, err error) {
	lenb, _ := reader.Peek(8)
	lengBuff := bytes.NewBuffer(lenb)
	var lens int64
	err = binary.Read(lengBuff, binary.LittleEndian, &lens)
	if err != nil {
		return
	}
	if int64(reader.Buffered()) < lens+8 {
		err = errors.New("lens:" + "bad")
		return
	}
	pack := make([]byte, int(8+lens))
	_, err = reader.Read(pack)
	if err != nil {
		return
	}
	return pack[8:], nil
}
func Println(conn net.Conn, i ...any) {
	var iList []any
	iList = append(iList, conn.LocalAddr(), "->", conn.RemoteAddr())
	for _, one := range i {
		iList = append(iList, one)
	}
	log.Println(iList...)
}
