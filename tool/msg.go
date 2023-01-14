package tool

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
)

var debug = false

// bytes size = 4096
// data len 4004=4096-8-60-8-16
const BufferSize = 4096

const versionSize = 16
const lenSize = 8
const hashSize = 60
const numSize = 8

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

func getHeaderSize() int {
	return versionSize + lenSize + hashSize + numSize
}

func NewKey(key string) Key {
	return Key{
		key:  key,
		keyB: []byte(key),
	}
}

func (k *Key) GetRawKey() string {
	return k.key
}

func (k *Key) Encode(i interface{}) (b [][]byte) {
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
	var bs [][]byte
	size := BufferSize - getHeaderSize()
	//size := BufferSize - versionSize - lenSize - hashSize - numSize
	lens := len(bk)
	for j := 0; j < lens; j += size {
		next := j + size
		if lens <= next {
			next = lens
		}
		bs = append(bs, bk[j:next])
	}
	return k.assemblyBytes(bs)
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

func (k *Key) GetMsg(reader *bufio.Reader) (c ConnMsg, err error) {
	var msg ConnMsg
	var b []byte
	for {
		b0, err1 := k.GetMsgV2(reader)
		if err1 != nil {
			if err1 == errWaitPack {
				//if err1.Error() == "wait pack" {
				b = append(b, b0...)
				i, err2 := reader.Discard(getHeaderSize() + len(b0))
				if err2 != nil {
					log.Println(i, err2)
					return msg, err2
				}
				continue
			} else {
				err = err1
				return
			}
		}
		b = append(b, b0...)
		i, err2 := reader.Discard(getHeaderSize() + len(b0))
		if err2 != nil {
			log.Println(i, err2)
			return msg, err2
		}
		break
	}
	err = k.Decode(&msg, b)
	if err != nil {
		return
	}
	m, ok := msg.Data.(map[string]any)
	if ok && m[mashBytesTag] != nil {
		msg.Data = MustBase64ToBytes(m[mashBytesTag].(string))
	}
	c = msg
	return
}

func (k *Key) GetMsgV2(reader *bufio.Reader) (b []byte, err error) {
	ver, err := reader.Peek(versionSize)
	if err != nil {
		return nil, err
	}
	if string(ver) != version {
		return nil, errors.New("the protocol is not go-CFC : is not " + version)
	}
	lenb, err := reader.Peek(versionSize + lenSize)
	if err != nil {
		return nil, err
	}
	lengBuff := bytes.NewBuffer(lenb[versionSize : versionSize+lenSize])
	var lens int64
	err = binary.Read(lengBuff, binary.LittleEndian, &lens)
	if err != nil {
		return
	}
	if lens <= int64(getHeaderSize()) {
		err = fmt.Errorf("lens: too small to %v  bytes", getHeaderSize())
		return
	}
	if lens > BufferSize {
		err = fmt.Errorf("lens: too long to %v bytes", BufferSize)
		return
	}
	//if int64(reader.Buffered()) != lens {
	//	fmt.Println("kkkk ", reader.Buffered(), lens)
	//	err = errors.New("lens:" + "bad")
	//	return
	//}
	pack, err := reader.Peek(int(lens))
	if err != nil {
		return
	}
	h, err := Decrypt(pack[versionSize+lenSize:versionSize+lenSize+hashSize], k.keyB)
	if err != nil {
		return
	}
	if !checkHash(h, pack[versionSize+lenSize+hashSize:]) {
		return nil, errors.New("hash check failed")
	}
	lengBuff2 := bytes.NewBuffer(pack[16+8+60 : 16+8+60+8])
	var num int64
	err = binary.Read(lengBuff2, binary.LittleEndian, &num)
	if err != nil {
		return
	}
	if num != 0 {
		return pack[getHeaderSize():], errWaitPack
	}
	return pack[getHeaderSize():], nil
}

func (k *Key) SetMsg(header, id string, code int, data interface{}) [][]byte {
	b, ok := data.([]byte)
	if ok {
		data = MustBytesToBase64(b)
	}
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
