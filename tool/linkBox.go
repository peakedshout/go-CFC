package tool

import (
	"bufio"
	"io"
	"sync"
)

type LinkBoxRWFn func(reader *bufio.Reader) ([][]byte, error)
type LinkBox struct {
	linkRW io.ReadWriter

	linkReader    *bufio.Reader
	linkReadFn    func(reader *bufio.Reader) ([][]byte, error) //link <- other
	linkWriteFn   func(reader *bufio.Reader) ([][]byte, error) //link -> other
	linkWriteLock sync.Mutex
}

func NewLinkBox(rw io.ReadWriter, readrSize int, readFn LinkBoxRWFn, writeFn LinkBoxRWFn) *LinkBox {
	lb := LinkBox{
		linkRW:        rw,
		linkReader:    bufio.NewReaderSize(rw, readrSize),
		linkReadFn:    readFn,
		linkWriteFn:   writeFn,
		linkWriteLock: sync.Mutex{},
	}
	return &lb
}

// ReadLinkBoxToWriter link -> other
func (lb *LinkBox) ReadLinkBoxToWriter(writer io.Writer, lock *sync.Mutex) error {
	if lb.linkWriteFn != nil {
		bs, err := lb.linkWriteFn(lb.linkReader)
		if err != nil {
			return err
		}
		if lock != nil {
			lock.Lock()
			defer lock.Unlock()
		}
		for _, one := range bs {
			_, err = writer.Write(one)
			if err != nil {
				return err
			}
		}
		return nil
	} else {
		buf := make([]byte, lb.linkReader.Size())
		n, err := lb.linkReader.Read(buf)
		if err != nil {
			return err
		}
		if lock != nil {
			lock.Lock()
			defer lock.Unlock()
		}
		_, err = writer.Write(buf[:n])
		if err != nil {
			return err
		}
		return nil
	}
}

// WriteLinkBoxFromReader link <- other
func (lb *LinkBox) WriteLinkBoxFromReader(reader *bufio.Reader) error {
	if lb.linkReadFn != nil {
		bs, err := lb.linkReadFn(reader)
		if err != nil {
			return err
		}
		return lb.writes(bs)
	} else {
		buf := make([]byte, reader.Size())
		n, err := reader.Read(buf)
		if err != nil {
			return err
		}
		return lb.write(buf[:n])
	}
}

func (lb *LinkBox) writes(bs [][]byte) error {
	lb.linkWriteLock.Lock()
	defer lb.linkWriteLock.Unlock()
	for _, one := range bs {
		_, err := lb.linkRW.Write(one)
		if err != nil {
			return err
		}
	}
	return nil
}
func (lb *LinkBox) write(b []byte) error {
	lb.linkWriteLock.Lock()
	defer lb.linkWriteLock.Unlock()
	_, err := lb.linkRW.Write(b)
	if err != nil {
		return err
	}
	return nil
}
