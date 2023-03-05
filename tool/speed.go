package tool

import (
	"fmt"
	"sync"
	"time"
)

type NetworkSpeedView struct {
	UploadSpeed       int
	DownloadSpeed     int
	UploadSpeedShow   string
	DownloadSpeedShow string
}

func CountAllNetworkSpeedView(list ...NetworkSpeedView) NetworkSpeedView {
	var resp NetworkSpeedView
	for _, one := range list {
		resp.UploadSpeed += one.UploadSpeed
		resp.DownloadSpeed += one.DownloadSpeed
	}
	resp.UploadSpeedShow = formatSpeed(resp.UploadSpeed)
	resp.DownloadSpeedShow = formatSpeed(resp.DownloadSpeed)
	return resp
}

func NewNetworkSpeedTicker() NetworkSpeedTicker {
	return NetworkSpeedTicker{
		Upload:   &SpeedTicker{},
		Download: &SpeedTicker{},
	}
}

type NetworkSpeedTicker struct {
	Upload   *SpeedTicker
	Download *SpeedTicker
}

func (n *NetworkSpeedTicker) ToView() NetworkSpeedView {
	u := n.Upload.Get()
	d := n.Download.Get()
	return NetworkSpeedView{
		UploadSpeed:       u,
		DownloadSpeed:     d,
		UploadSpeedShow:   formatSpeed(u),
		DownloadSpeedShow: formatSpeed(d),
	}
}

type SpeedTicker struct {
	read  int
	write int
	rLock sync.Mutex
	wLock sync.Mutex
	t     time.Time
}

func (st *SpeedTicker) Set(i int) {
	st.wLock.Lock()
	defer st.wLock.Unlock()
	t := time.Now().Round(time.Second)
	if !st.t.Equal(t) {
		st.rLock.Lock()
		st.read = st.write
		st.write = i
		st.t = t
		st.rLock.Unlock()
	} else {
		st.write += i
	}
}
func (st *SpeedTicker) Get() int {
	st.rLock.Lock()
	defer st.rLock.Unlock()
	t := time.Now().Round(time.Second)
	s := t.Sub(st.t)
	if s > time.Second {
		if s <= 2*time.Second {
			st.wLock.Lock()
			st.read = st.write
			st.write = 0
			st.t = t
			st.wLock.Unlock()
		}
		st.read = 0
	}
	return st.read
}

func formatSpeed(speed int) (size string) {
	fileSize := int64(speed)
	if fileSize < 1024 {
		return fmt.Sprintf("%.2fB/s", float64(fileSize)/float64(1))
	} else if fileSize < (1024 * 1024) {
		return fmt.Sprintf("%.2fKB/s", float64(fileSize)/float64(1024))
	} else if fileSize < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fMB/s", float64(fileSize)/float64(1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fGB/s", float64(fileSize)/float64(1024*1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fTB/s", float64(fileSize)/float64(1024*1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2fEB/s", float64(fileSize)/float64(1024*1024*1024*1024*1024))
	}
}
