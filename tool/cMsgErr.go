package tool

import (
	"errors"
	"fmt"
	"strings"
)

var ErrKeyIsNot32Bytes = errors.New("key is not 32bytes")
var ErrNameIsNil = errors.New("name is nil")

var ErrReadCSkipToFastConn = errors.New("skip to fastConn")
var ErrReadCProtocolIsNotGoCFC = errors.New("the protocol is not go-CFC : is not " + version)
var ErrReadCMsgLensTooShort = fmt.Errorf("lens: too small to %v  bytes", getHeaderSize())
var ErrReadCMsgLensTooLong = fmt.Errorf("lens: too long to %v bytes", BufferSize)
var ErrReadCMsgHashCheckFailed = errors.New("hash check failed")
var ErrReadCMsgWaitPack = errors.New("wait pack")

var ErrHandleCMsgMissProxyClient = errors.New("not Not found proxy client ")
var ErrHandleCMsgMissProxyTaskRoom = errors.New("not Not found proxy task room ")

var ErrHandleCMsgProxyClientNameIsNil = errors.New("need one proxy client name to register")

var ErrTimeout = errors.New("timeout")

var ErrIsDisable = errors.New("is disable")

var ErrHandshakeIsDad = errors.New("handshake is bad")

var ErrReqUnexpectedHeader = errors.New("unexpected resp header")
var ErrReqBadAny = func(a ...any) error { return fmt.Errorf("req bad : %v", a) }

var ErrConnIsNil = errors.New("conn is nil")

var ErrOpenSubUnexpectedOdj = errors.New("unexpected resp Odj")
var ErrOpenSubBoxBadAny = func(a ...any) error { return fmt.Errorf("open sub box bad : %v", a) }

var ErrBoxIsNil = errors.New("box is nil")
var ErrBoxIsClosed = errors.New("box is closed")
var ErrSubIsDisable = errors.New("sub box is disable")
var ErrBoxComplexListen = errors.New("box complex listen")
var ErrBoxStopListen = errors.New("box stop listen")

//var ErrHandleCMsgBad = errors.New("need one proxy client name to register")

func ErrAppend(err error, errs ...error) error {
	if err == nil {
		return err
	}
	rawErr := err.Error()

	var hasErrsList []string
	for _, one := range errs {
		if one != nil {
			hasErrsList = append(hasErrsList, one.Error())
		}
	}
	if len(hasErrsList) > 0 {
		rawErr += " : [ " + strings.Join(hasErrsList, " | ") + " ]"
	}
	return errors.New(rawErr)
}

var errWaitPack = errors.New("wait pack")
