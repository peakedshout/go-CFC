package tool

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"time"
)

var debug = false

type ConnMsg struct {
	Header string
	Code   int
	Data   interface{}
	Id     string
}
type Key struct {
	key  string
	keyB []byte
}
type Ping struct {
	Ping time.Duration
}

func NewKey(key string) Key {
	return Key{
		key:  key,
		keyB: []byte(key),
	}
}
func (k *Key) Encode(i interface{}) (b []byte) {
	var bk []byte
	if !debug {
		bs, err := Encrypt(MustMarshal(i), k.keyB)
		if err != nil {
			panic(err)
		}
		bk = bs
	} else {
		bk = MustMarshal(i)
	}
	lens := int64(len(bk))
	var pkg = new(bytes.Buffer)
	err := binary.Write(pkg, binary.LittleEndian, lens)
	if err != nil {
		panic(err)
	}
	err = binary.Write(pkg, binary.LittleEndian, bk)
	if err != nil {
		panic(err)
	}
	b = pkg.Bytes()
	return
}
func (k *Key) Decode(i interface{}, data []byte) error {
	if len(data) == 0 {
		return errors.New("nil")
	}
	var b []byte
	if !debug {
		bs, err := Decrypt(data, k.keyB)
		if err != nil {
			return err
		}
		b = bs
	} else {
		b = data
	}
	return json.Unmarshal(b, &i)
}

//	func (k *Key) GetMsg(b []byte) ConnMsg {
//		var msg ConnMsg
//		k.Decode(&msg, b)
//		return msg
//	}
func (k *Key) GetMsg(reader *bufio.Reader) (c ConnMsg, err error) {
	var msg ConnMsg
	b, err := k.GetMsgV2(reader)
	if err != nil {
		return
	}
	err = k.Decode(&msg, b)
	if err != nil {
		return
	}
	c = msg
	return
}

func (k *Key) GetMsgV2(reader *bufio.Reader) (b []byte, err error) {
	lenb, err := reader.Peek(8)
	if err != nil {

		return nil, err
	}
	lengBuff := bytes.NewBuffer(lenb)
	var lens int64
	err = binary.Read(lengBuff, binary.LittleEndian, &lens)
	if err != nil {
		return
	}
	if lens <= 0 {
		err = errors.New("lens:" + "A negative number")
		return
	}
	if lens >= 2097152 { //~2MB
		err = errors.New("lens:" + "too long to 2MB")
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

func (k *Key) SetMsg(header, id string, code int, data interface{}) []byte {
	return k.Encode(ConnMsg{
		Header: header,
		Code:   code,
		Data:   data,
		Id:     id,
	})
}

type OdjClientInfo struct {
	Name string
}

type OdjAddr struct {
	Id   string
	Addr string
}

type OdjMsg struct {
	Msg string
}

type OdjInfo struct {
	Id       string
	User     string
	Password string
}

type OdjSub struct {
	SrcName string
	DstKey  string
}

type OdjIdList struct {
	IdList []string
}
type OdjPing struct {
	Name   string
	Ping   Ping
	Active bool
}
