package client

import "github.com/peakedshout/go-CFC/tool"

type SubType string

const (
	SubTypeUP2P  = SubType("SubTypeUP2P")
	SubTypeP2P   = SubType("SubTypeP2P")
	SubTypeProxy = SubType("SubTypeProxy")
)

func (st *SubType) String() string {
	return string(*st)
}

func (box *DeviceBox) GetSubBoxBySubType(name string, subType SubType) (*SubBox, error) {
	switch subType {
	case SubTypeProxy:
		return box.GetSubBox(name)
	case SubTypeP2P:
		return box.GetSubBoxByP2P(name)
	case SubTypeUP2P:
		return box.GetSubBoxByUP2P(name)
	default:
		return nil, tool.ErrSubTypeInvalid
	}
}

func (box *DeviceBox) GetSubBoxBySubTypeMix(name string, subTypes []SubType) (*SubBox, []error, error) {
	errs := make([]error, len(subTypes))
	for i, one := range subTypes {
		sub, err := box.GetSubBoxBySubType(name, one)
		if err != nil {
			errs[i] = err
			continue
		}
		return sub, errs, nil
	}
	return nil, errs, tool.ErrSubTypeToMixGetSubBoxFailed
}
