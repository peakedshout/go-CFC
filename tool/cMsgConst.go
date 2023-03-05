package tool

const version = `go-CFC-v01.02.00`

// bytes size = 4096
// data len 4004=4096-8-60-8-16
const (
	BufferSize  = 4096
	versionSize = 16
	lenSize     = 8
	hashSize    = 60
	numSize     = 8
)

func getHeaderSize() int {
	return versionSize + lenSize + hashSize + numSize
}

const (
	P2PTcpReset   = "P2PTcpReset"
	P2PTcpStart   = "P2PTcpStart"
	P2PTCpSucceed = "P2PTCpSucceed"
	P2PTCpFailed  = "P2PTCpSucceed"
)
