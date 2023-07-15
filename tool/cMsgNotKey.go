package tool

import (
	"encoding/json"
	"io"
)

func ReadCMsgNotKeyJson2(reader io.Reader) (cMsg ConnMsg, err error) {
	d := json.NewDecoder(reader)
	err = d.Decode(&cMsg)
	return cMsg, err
}

func MustReadCMsgNotKeyJson2(reader io.Reader) (cMsg ConnMsg) {
	cMsg, err := ReadCMsgNotKeyJson2(reader)
	if err != nil {
		panic(err)
	}
	return cMsg
}

func ReadCMsgNotKeyJson(b []byte) (cMsg ConnMsg, err error) {
	err = json.Unmarshal(b, &cMsg)
	return cMsg, err
}

func WriteCMsgNotKeyJson(cMsg ConnMsg) (b []byte, err error) {
	return json.Marshal(cMsg)
}

func MustReadCMsgNotKeyJson(b []byte) (cMsg ConnMsg) {
	cMsg, err := ReadCMsgNotKeyJson(b)
	if err != nil {
		panic(err)
	}
	return cMsg
}

func MustWriteCMsgNotKeyJson(cMsg ConnMsg) (b []byte) {
	b, err := WriteCMsgNotKeyJson(cMsg)
	if err != nil {
		panic(err)
	}
	return b
}
