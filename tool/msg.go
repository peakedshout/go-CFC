package tool

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"
)

var debug = false

// bytes size = 4096
// data len 4004=4096-8-60-8-16
const BufferSize = 4096

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
	size := BufferSize - 8 - 60 - 8 - 16
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
			if err1.Error() == "wait pack" {
				b = append(b, b0...)
				i, err2 := reader.Discard(16 + 8 + 60 + 8 + len(b0))
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
		i, err2 := reader.Discard(16 + 8 + 60 + 8 + len(b0))
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
	c = msg
	return
}

func (k *Key) GetMsgV2(reader *bufio.Reader) (b []byte, err error) {
	ver, err := reader.Peek(16)
	if err != nil {
		return nil, err
	}
	if string(ver) != version {
		return nil, errors.New("the protocol is not go-CFC : is not " + version)
	}
	lenb, err := reader.Peek(8 + 16)
	if err != nil {
		return nil, err
	}
	lengBuff := bytes.NewBuffer(lenb[16:24])
	var lens int64
	err = binary.Read(lengBuff, binary.LittleEndian, &lens)
	if err != nil {
		return
	}
	if lens <= 8+60+8 {
		err = errors.New("lens:" + "too small to " + strconv.Itoa(0+60+8) + " bytes")
		return
	}
	if lens > BufferSize {
		err = errors.New("lens:" + "too long to " + strconv.Itoa(BufferSize) + " bytes")
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
	h, err := Decrypt(pack[16+8:16+8+60], k.keyB)
	if err != nil {
		return
	}
	if !checkHash(h, pack[16+8+60:]) {
		return nil, errors.New("hash check failed")
	}
	lengBuff2 := bytes.NewBuffer(pack[16+8+60 : 16+8+60+8])
	var num int64
	err = binary.Read(lengBuff2, binary.LittleEndian, &num)
	if err != nil {
		return
	}
	if num != 0 {
		return pack[16+8+60+8:], errors.New("wait pack")
	}
	return pack[16+8+60+8:], nil
}

func (k *Key) SetMsg(header, id string, code int, data interface{}) [][]byte {
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
