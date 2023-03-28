package complexRW

import (
	"errors"
	"io"
	"sync"
)

type RWId int

type RWConference struct {
}

type RWBroadcast struct {
	spokesman chan []byte
	listener  map[RWId]io.Writer
	count     RWId
	wLock     sync.Mutex
	closer    sync.Once
	stop      chan uint8
}

func NewRWBroadcast(rw io.Reader, size int) *RWBroadcast {
	rwb := &RWBroadcast{
		spokesman: make(chan []byte, size),
		listener:  make(map[RWId]io.Writer),
		count:     0,
		wLock:     sync.Mutex{},
		closer:    sync.Once{},
		stop:      make(chan uint8, 1),
	}
	if rw != nil {
		go func() {
			for {
				buff := make([]byte, size)
				n, err := rw.Read(buff)
				if err != nil {
					return
				}
				rwb.Write(buff[:n])
			}
		}()
	}
	go func() {
		for {
			select {
			case <-rwb.stop:
				rwb.stop <- 1
				return
			case b := <-rwb.spokesman:
				rwb.allRWCopy(b)
			}
		}
	}()
	return rwb
}

func (rwb *RWBroadcast) Close() {
	rwb.closer.Do(func() {
		rwb.stop <- 1
	})
}

func (rwb *RWBroadcast) Write(b []byte) (n int, err error) {
	select {
	case <-rwb.stop:
		rwb.stop <- 1
		return 0, errors.New("is closed")
	case rwb.spokesman <- b:
		return len(b), nil
	}
}

func (rwb *RWBroadcast) SetListener(ln io.Writer) RWId {
	rwb.wLock.Lock()
	defer rwb.wLock.Unlock()
	rwb.count++
	rwb.listener[rwb.count] = ln
	return rwb.count
}
func (rwb *RWBroadcast) DelListener(id RWId) {
	delete(rwb.listener, id)
}

func (rwb *RWBroadcast) allRWCopy(b []byte) {
	rwb.wLock.Lock()
	defer rwb.wLock.Unlock()
	for _, one := range rwb.listener {
		one.Write(b)
	}
}
