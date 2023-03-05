package tool

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
)

var debug = false

type Key struct {
	key  string
	keyB []byte
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

func (k *Key) assemblyBytes(bs [][]byte) [][]byte {
	var bo [][]byte
	l := len(bs)
	for i, b1 := range bs {
		lens := int64(len(b1) + getHeaderSize()) //ver len hash num... data
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
		if err != nil || len(h2) != hashSize {
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
