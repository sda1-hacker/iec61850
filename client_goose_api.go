package iec61850

import "time"

type GooseManager struct {
	Iface    string         // 网卡的名称
	GoCbRef  string         // GOOSE控制快的ref --> simpleIOGenericIO/LLN0$GO$gcbEvents
	dataChan chan GooseData // 数据channel
}

type GooseData struct {
	AppId     string
	SrcMac    string
	DstMac    string
	GoId      string
	GoCbRef   string
	Timestamp time.Time
	Entries   []GooseDataEntry
}

// 订阅的数据
type GooseDataEntry struct {
	Name   string
	Value  MmsValue
	Reason int
}

func NewGooseManager(iface string, goCbRef string, channelSize int) (*GooseManager, error) {
	if channelSize <= 0 {
		channelSize = 100
	}
	return &GooseManager{
		Iface:    iface,
		GoCbRef:  goCbRef,
		dataChan: make(chan GooseData, channelSize),
	}, nil
}
