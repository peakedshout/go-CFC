package tool

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/peakedshout/go-CFC/loger"
	"sync/atomic"
)

func (k *Key) ReadCMsg(reader *bufio.Reader, skip *atomic.Bool, speed *SpeedTicker) (cMsg ConnMsg, err error) {
	var msg ConnMsg
	var b []byte
	for {
		b0, err1 := k.ReadCPacket(reader, skip)
		if err1 != nil {
			if err1 == errWaitPack {
				b = append(b, b0...)
				n, err2 := reader.Discard(getHeaderSize() + len(b0))
				if err2 != nil {
					loger.SetLogDebug(err2)
					return msg, err2
				}
				if speed != nil {
					speed.Set(n)
				}
				continue
			} else {
				return msg, err1
			}
		}
		b = append(b, b0...)
		n, err2 := reader.Discard(getHeaderSize() + len(b0))
		if err2 != nil {
			loger.SetLogDebug(err2)
			return msg, err2
		}
		if speed != nil {
			speed.Set(n)
		}
		break
	}
	err = k.Decode(&msg, b)
	if err != nil {
		loger.SetLogDebug(err)
		return msg, err
	}
	m, ok := msg.Data.(map[string]any)
	if ok && m[mashBytesTag] != nil {
		msg.Data = MustBase64ToBytes(m[mashBytesTag].(string))
	}
	return msg, nil
}
func (k *Key) ReadCPacket(reader *bufio.Reader, skip *atomic.Bool) (b []byte, err error) {
	_, err = reader.Peek(1)
	if err != nil {
		loger.SetLogDebug(err)
		return nil, err
	}
	if skip != nil && skip.Load() {
		err = ErrReadCSkipToFastConn
		loger.SetLogDebug(err)
		return nil, err
	}
	ver, err := reader.Peek(versionSize)
	if err != nil {
		loger.SetLogDebug(err)
		return nil, err
	}
	if string(ver) != version {
		err = ErrReadCProtocolIsNotGoCFC
		loger.SetLogDebug(err)
		return nil, err
	}
	lenb, err := reader.Peek(versionSize + lenSize)
	if err != nil {
		loger.SetLogDebug(err)
		return nil, err
	}
	lengBuff := bytes.NewBuffer(lenb[versionSize : versionSize+lenSize])
	var lens int64
	err = binary.Read(lengBuff, binary.LittleEndian, &lens)
	if err != nil {
		loger.SetLogDebug(err)
		return nil, err
	}
	if lens <= int64(getHeaderSize()) {
		err = ErrReadCMsgLensTooShort
		loger.SetLogDebug(err)
		return nil, err
	}
	if lens > BufferSize {
		err = ErrReadCMsgLensTooLong
		loger.SetLogDebug(err)
		return nil, err
	}
	pack, err := reader.Peek(int(lens))
	if err != nil {
		loger.SetLogDebug(err)
		return nil, err
	}
	h, err := Decrypt(pack[versionSize+lenSize:versionSize+lenSize+hashSize], k.keyB)
	if err != nil {
		loger.SetLogDebug(err)
		return nil, err
	}
	if !checkHash(h, pack[versionSize+lenSize+hashSize:]) {
		err = ErrReadCMsgHashCheckFailed
		loger.SetLogDebug(err)
		return nil, err
	}
	lengBuff2 := bytes.NewBuffer(pack[16+8+60 : 16+8+60+8])
	var num int64
	err = binary.Read(lengBuff2, binary.LittleEndian, &num)
	if err != nil {
		loger.SetLogDebug(err)
		return nil, err
	}
	if num != 0 {
		return pack[getHeaderSize():], errWaitPack
	}
	return pack[getHeaderSize():], nil
}
