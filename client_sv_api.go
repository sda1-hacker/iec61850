package iec61850

// #include "client_sv.h"
// #include <stdlib.h>
import "C"
import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
	"unsafe"
)

/**
| IEC 61850 type | required bytes |
*  | -------------- | -------------- |
*  | BOOLEAN        | 1 byte         |
*  | INT8           | 1 byte         |
*  | INT16          | 2 byte         |
*  | INT32          | 4 byte         |
*  | INT64          | 8 byte         |
*  | INT8U          | 1 byte         |
*  | INT16U         | 2 byte         |
*  | INT24U         | 3 byte         |
*  | INT32U         | 4 byte         |
*  | INT64U         | 8 byte         |
*  | FLOAT32        | 4 byte         |
*  | FLOAT64        | 8 byte         |
*  | ENUMERATED     | 4 byte         |
*  | CODED ENUM     | 4 byte         |
*  | OCTET STRING   | 20 byte        |
*  | VISIBLE STRING | 35 byte        |
*  | TimeStamp      | 8 byte         |
*  | EntryTime      | 6 byte         |
*  | BITSTRING      | 4 byte         |
*  | Quality        | 4 byte         |
**/

// ASDU 数据类型
type ASDU_DATA_TYPE int

const (
	// 1
	ASDU_DATATYPE_INT8 ASDU_DATA_TYPE = iota
	// 2
	ASDU_DATATYPE_INT16
	// 4
	ASDU_DATATYPE_INT32
	// 8
	ASDU_DATATYPE_INT64
	// 1
	ASDU_DATATYPE_UINT8
	// 2
	ASDU_DATATYPE_UINT16
	// 4
	ASDU_DATATYPE_UINT32
	// 8
	ASDU_DATATYPE_UINT64
	// 4
	ASDU_DATATYPE_FLOAT32
	// 8
	ASDU_DATATYPE_FLOAT64
	// 8
	ASDU_DATATYPE_TIMESTAMP
	// 4
	ASDU_DATATYPE_QUALITY
)

var (
	nextSvManagerID  uintptr
	svManagersMu     sync.RWMutex
	svManagers       = make(map[uintptr]*SvManager)
	dataTypeBytesNum = []int{1, 2, 4, 8, 1, 2, 4, 8, 4, 8, 8, 4} // 数据类型所占的byte数量，dataTypeBytesNum[ASDU_DATATYPE_QUALITY] 来获取对应的字节数
)

// ASDU数据的结构
type ASDUDataStruct struct {
	Index    int            // 索引
	DataType ASDU_DATA_TYPE // 数据类型
	Offset   int            // 计算后的字节偏移量 -- 系统计算，不需要用户填写
}

type SvManager struct {
	Iface         string                      // 网卡的名称
	AppId         uint16                      // appId
	dataChan      chan SvData                 // 数据channel
	ctx           context.Context             // ctx
	cancelFunc    context.CancelFunc          // 停止函数
	id            uintptr                     // SvManager的id
	dataStructMap map[string][]ASDUDataStruct // key: SvID, value: []ASDUDataStruct
	wg            sync.WaitGroup              // WaitGroup 用于同步
}

type SvData struct {
	SvID    string
	SmpCnt  uint16
	ConfRev uint32
	RefrTm  time.Time // 参考时间
	Entries []SvDataEntry
}

type SvDataEntry struct {
	Index         int            // 字段的索引
	Asdu_DataType ASDU_DATA_TYPE // ASDU数据类型
	Value         interface{}    // 值
}

func NewSvManager(iface string, appId uint16, dataStructMap map[string][]ASDUDataStruct, channelSize int) *SvManager {
	if channelSize <= 0 {
		channelSize = 100
	}
	ctx, cancelFunc := context.WithCancel(context.Background())

	// 预处理数据结构，计算偏移量
	preparedMap := make(map[string][]ASDUDataStruct)
	for svID, structs := range dataStructMap {
		// 复制结构体
		copyStructs := make([]ASDUDataStruct, len(structs))
		copy(copyStructs, structs)

		// 按索引排序
		sort.Slice(copyStructs, func(i, j int) bool {
			return copyStructs[i].Index < copyStructs[j].Index
		})

		// 计算偏移量
		offset := 0
		for i := range copyStructs {
			copyStructs[i].Offset = offset
			offset += dataTypeBytesNum[copyStructs[i].DataType]
		}
		preparedMap[svID] = copyStructs
	}

	return &SvManager{
		Iface:         iface,
		AppId:         appId,
		dataChan:      make(chan SvData, channelSize),
		ctx:           ctx,
		cancelFunc:    cancelFunc,
		dataStructMap: preparedMap,
	}
}

func (m *SvManager) Subscribe() {
	m.wg.Add(1)
	defer m.wg.Done() // 确保方法退出时标记完成

	// 创建接收器
	receiver := C.SVReceiver_create()

	//m.receiver = receiver

	// 设置网络接口（默认eth0）
	cIface := C.CString(m.Iface)
	defer C.free(unsafe.Pointer(cIface))
	C.SVReceiver_setInterfaceId(receiver, cIface)

	// 创建订阅者（APPID 0x4000）
	subscriber := C.SVSubscriber_create(nil, C.uint16_t(m.AppId))
	// m.subscriber = subscriber

	// 在全局注册表中注册管理器
	svManagersMu.Lock()
	m.id = nextSvManagerID
	nextSvManagerID++
	svManagers[m.id] = m
	svManagersMu.Unlock()

	// 设置回调函数
	C.SVSubscriber_setListener(subscriber, C.SvCallback(C.svCallbackFunction), unsafe.Pointer(m.id))

	// 添加订阅者到接收器
	C.SVReceiver_addSubscriber(receiver, subscriber)

	// 启动接收器
	C.SVReceiver_start(receiver)

	log.Printf("listen iface: %s \n", m.Iface)

	<-m.ctx.Done()
	log.Printf("cancel sv \n")

	// 结束之后销毁C资源
	C.SVReceiver_stop(receiver)
	C.SVReceiver_destroy(receiver)
	// C.SVSubscriber_destroy(subscriber) // 官方的案例，没有调用这个方法
	log.Printf("回收sv资源.. \n")

}

func (m *SvManager) UnSubscribe() {

	// 取消上下文
	m.cancelFunc()

	m.wg.Wait()

	// 从全局注册表移除
	svManagersMu.Lock()
	delete(svManagers, m.id)
	svManagersMu.Unlock()

	// 安全关闭channel
	close(m.dataChan)

}

func (m *SvManager) GetDataChan() <-chan SvData {
	return m.dataChan
}

//export goSvCallback
func goSvCallback(subscriber C.SVSubscriber, parameter unsafe.Pointer, asdu C.SVSubscriber_ASDU) {

	// 从参数中获取ID
	id := uintptr(parameter)

	// 从全局注册表获取管理器
	svManagersMu.Lock()
	manager := svManagers[id]
	svManagersMu.Unlock()

	// 获取基本参数
	svID := C.GoString(C.SVSubscriber_ASDU_getSvId(asdu))
	smpCnt := C.SVSubscriber_ASDU_getSmpCnt(asdu)
	confRev := C.SVSubscriber_ASDU_getConfRev(asdu)

	// 获取参考时间（如果存在）
	var refrTm time.Time
	if C.SVSubscriber_ASDU_hasRefrTm(asdu) {
		nsTime := C.SVSubscriber_ASDU_getRefrTmAsNs(asdu)
		refrTm = time.Unix(0, int64(nsTime))
	}

	svStructs, ok := manager.dataStructMap[svID]
	if !ok {
		log.Printf("No data structure defined for SvID: %s", svID)
		return
	}

	// 获取数据总大小
	totalSize := int(C.SVSubscriber_ASDU_getDataSize(asdu))
	data := SvData{
		SvID:    svID,
		SmpCnt:  uint16(smpCnt),
		ConfRev: uint32(confRev),
		RefrTm:  refrTm,
		Entries: make([]SvDataEntry, 0, len(svStructs)),
	}

	// 解析数据
	for _, svStruct := range svStructs {
		// 检查偏移量是否有效
		requiredBytes := dataTypeBytesNum[svStruct.DataType]
		if svStruct.Offset+requiredBytes > totalSize {
			log.Printf("Offset %d out of range for SvID %s (total size: %d)",
				svStruct.Offset, svID, totalSize)
			continue
		}

		value, err := getGoValue(asdu, svStruct.Offset, svStruct.DataType)
		if err != nil {
			log.Printf("Error parsing value at offset %d for SvID %s: %v",
				svStruct.Offset, svID, err)
			continue
		}

		data.Entries = append(data.Entries, SvDataEntry{
			Index:         svStruct.Index,
			Asdu_DataType: svStruct.DataType,
			Value:         value,
		})
	}

	// 发送数据到通道
	select {
	case manager.dataChan <- data:
	default:
		log.Println("SV channel full, dropping data")
	}

}

func getGoValue(asdu C.SVSubscriber_ASDU, offset int, asduDataType ASDU_DATA_TYPE) (interface{}, error) {
	switch asduDataType {
	case ASDU_DATATYPE_INT8:
		return int8(C.SVSubscriber_ASDU_getINT8(asdu, C.int(offset))), nil
	case ASDU_DATATYPE_INT16:
		return int16(C.SVSubscriber_ASDU_getINT16(asdu, C.int(offset))), nil
	case ASDU_DATATYPE_INT32:
		return int32(C.SVSubscriber_ASDU_getINT32(asdu, C.int(offset))), nil
	case ASDU_DATATYPE_INT64:
		return int64(C.SVSubscriber_ASDU_getINT64(asdu, C.int(offset))), nil
	case ASDU_DATATYPE_UINT8:
		return uint8(C.SVSubscriber_ASDU_getINT8U(asdu, C.int(offset))), nil
	case ASDU_DATATYPE_UINT16:
		return uint16(C.SVSubscriber_ASDU_getINT16U(asdu, C.int(offset))), nil
	case ASDU_DATATYPE_UINT32:
		return uint32(C.SVSubscriber_ASDU_getINT32U(asdu, C.int(offset))), nil
	case ASDU_DATATYPE_UINT64:
		return uint64(C.SVSubscriber_ASDU_getINT64U(asdu, C.int(offset))), nil
	case ASDU_DATATYPE_FLOAT32:
		return float32(C.SVSubscriber_ASDU_getFLOAT32(asdu, C.int(offset))), nil
	case ASDU_DATATYPE_FLOAT64:
		return float64(C.SVSubscriber_ASDU_getFLOAT64(asdu, C.int(offset))), nil
	case ASDU_DATATYPE_TIMESTAMP:
		ts := C.SVSubscriber_ASDU_getTimestamp(asdu, C.int(offset))
		return ts, nil
		// return time.Unix(0, int64(ts)*int64(time.Millisecond)), nil
	case ASDU_DATATYPE_QUALITY:
		q := C.SVSubscriber_ASDU_getQuality(asdu, C.int(offset))
		return parseQuality(uint32(q)), nil
	default:
		return nil, fmt.Errorf("unsupported type %d", asduDataType)
	}
}

// 解析质量位
func parseQuality(q uint32) map[string]bool {
	return map[string]bool{
		"validity":       (q & 0xC000) != 0,
		"overflow":       (q & 0x2000) != 0,
		"out_of_range":   (q & 0x1000) != 0,
		"bad_reference":  (q & 0x0800) != 0,
		"oscillatory":    (q & 0x0400) != 0,
		"failure":        (q & 0x0200) != 0,
		"old_data":       (q & 0x0100) != 0,
		"inconsistent":   (q & 0x0080) != 0,
		"inaccurate":     (q & 0x0040) != 0,
		"source":         (q & 0x0020) != 0,
		"test":           (q & 0x0010) != 0,
		"operator_block": (q & 0x0008) != 0,
		"derived":        (q & 0x0004) != 0,
		"elapsed_time":   (q & 0x0002) != 0,
	}
}
