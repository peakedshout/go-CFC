package tool

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
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

func toolHash(b []byte) []byte {
	s := sha256.New()
	s.Write(b)
	return s.Sum(nil)
}

func (k *Key) assemblyBytes(bs [][]byte) [][]byte {
	var bo [][]byte
	l := len(bs)
	for i, b1 := range bs {
		lens := int64(len(b1) + 8 + 60 + 8 + 16) //ver len hash num... data
		var pkg = new(bytes.Buffer)
		//version
		err := binary.Write(pkg, binary.LittleEndian, []byte(version))
		if err != nil {
			panic(err)
		}
		//len
		err = binary.Write(pkg, binary.LittleEndian, lens)
		if err != nil {
			panic(err)
		}

		//num+data
		var pkg2 = new(bytes.Buffer)
		err = binary.Write(pkg2, binary.LittleEndian, int64(l-i-1))
		if err != nil {
			panic(err)
		}
		err = binary.Write(pkg2, binary.LittleEndian, b1)
		if err != nil {
			panic(err)
		}
		b2 := pkg2.Bytes()
		h := toolHash(b2)
		h2, err := Encrypt(h, k.keyB)
		if err != nil || len(h2) != 60 {
			panic(err)
		}
		//hash
		err = binary.Write(pkg, binary.LittleEndian, h2)
		if err != nil {
			panic(err)
		}
		//num + data
		err = binary.Write(pkg, binary.LittleEndian, b2)
		if err != nil {
			panic(err)
		}
		bo = append(bo, pkg.Bytes())
	}
	return bo
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
