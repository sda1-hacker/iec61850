package iec61850

// #include "client_goose.h"
// #include <stdlib.h>
import "C"
import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"unsafe"
)

var (
	nextGooseManagerID uintptr
	gooseManagersMu    sync.RWMutex
	gooseManagers      = make(map[uintptr]*GooseManager)
)

type GooseManager struct {
	Iface      string             // 网卡的名称
	GoCbRef    string             // GOOSE控制快的ref --> simpleIOGenericIO/LLN0$GO$gcbAnalogValues，这个必须要指定
	AppId      int                // appid
	dataChan   chan GooseData     // 数据channel
	ctx        context.Context    // ctx
	cancelFunc context.CancelFunc // 停止函数
	id         uintptr            // GooseManager的id

	// 添加C资源指针
	receiver   C.GooseReceiver
	subscriber C.GooseSubscriber
}

type GooseData struct {
	AppId     int32
	GoId      string
	GoCbRef   string
	StNum     uint
	SqNum     uint
	Timestamp time.Time
	Entries   []GooseDataEntry
}

// 订阅的数据
type GooseDataEntry struct {
	Index int      // 数据所在的索引位置
	Value MmsValue // MMS值
}

func NewGooseManager(iface string, goCbRef string, appId int, channelSize int) *GooseManager {
	if channelSize <= 0 {
		channelSize = 100
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	return &GooseManager{
		Iface:      iface,
		GoCbRef:    goCbRef,
		AppId:      appId,
		dataChan:   make(chan GooseData, channelSize),
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}
}

// 订阅
func (m *GooseManager) Subscribe() {
	// 创建接收器
	receiver := C.GooseReceiver_create()
	m.receiver = receiver

	// 设置网络接口（默认eth0）
	iface := C.CString(m.Iface)
	defer C.free(unsafe.Pointer(iface))
	C.GooseReceiver_setInterfaceId(receiver, iface)

	// 创建订阅者 -- 这个需要
	gocbRef := C.CString(m.GoCbRef)
	defer C.free(unsafe.Pointer(gocbRef))
	subscriber := C.GooseSubscriber_create(gocbRef, nil)
	m.subscriber = subscriber

	// 设置目标MAC地址 (01:0c:cd:01:00:01)  --> 这个需要抽离出来作为参数
	var dstMac [6]C.uint8_t = [6]C.uint8_t{0x01, 0x0c, 0xcd, 0x01, 0x00, 0x01}
	C.GooseSubscriber_setDstMac(subscriber, &dstMac[0])

	// 设置AppID
	C.GooseSubscriber_setAppId(subscriber, C.uint16_t(m.AppId))

	// 在全局注册表中注册管理器
	gooseManagersMu.Lock()
	m.id = nextGooseManagerID
	nextGooseManagerID++
	gooseManagers[m.id] = m
	gooseManagersMu.Unlock()

	// 设置回调
	C.GooseSubscriber_setListener(subscriber, C.GooseCallback(C.gooseCallbackFunction), unsafe.Pointer(m.id))

	// 添加订阅者到接收器
	C.GooseReceiver_addSubscriber(receiver, subscriber)

	// 启动接收器
	C.GooseReceiver_start(receiver)

	log.Printf("开启监听网卡: %s \n", m.Iface)

	// 持续监听
	//for {
	//	select {
	//	case <-m.ctx.Done():
	//		// 清理资源
	//		C.GooseReceiver_stop(receiver)
	//		C.GooseSubscriber_destroy(subscriber)
	//		C.GooseReceiver_destroy(receiver)
	//		log.Printf("取消GOOSE监听..")
	//		return
	//	}
	//}
	<-m.ctx.Done()
	log.Printf("取消GOOSE监听..")

}

// 取消订阅
func (m *GooseManager) UnSubscribe() {
	// 先停止再清理
	if m.receiver != nil {
		C.GooseReceiver_stop(m.receiver)
	}

	// 从全局注册表中移除
	if m.id != 0 {
		gooseManagersMu.Lock()
		delete(gooseManagers, m.id)
		gooseManagersMu.Unlock()
		m.id = 0
	}

	// 清理C资源
	if m.subscriber != nil {
		C.GooseSubscriber_destroy(m.subscriber)
		m.subscriber = nil
	}

	if m.receiver != nil {
		C.GooseReceiver_destroy(m.receiver)
		m.receiver = nil
	}

	// 取消上下文
	m.cancelFunc()

	// 安全关闭channel
	close(m.dataChan)
}

// 获取数据channel
func (m *GooseManager) GetDataChan() <-chan GooseData {
	return m.dataChan
}

//export goGooseCallback
func goGooseCallback(subscriber C.GooseSubscriber, parameter unsafe.Pointer) {

	// 从参数中获取ID
	id := uintptr(parameter)

	// 从全局注册表获取管理器
	gooseManagersMu.Lock()
	manager := gooseManagers[id]
	gooseManagersMu.Unlock()

	// 获取数据集值
	values := C.GooseSubscriber_getDataSetValues(subscriber)
	timestamp := uint64(C.GooseSubscriber_getTimestamp(subscriber))
	data := GooseData{
		AppId:     int32(C.GooseSubscriber_getAppId(subscriber)),
		GoId:      C.GoString(C.GooseSubscriber_getGoId(subscriber)),
		GoCbRef:   C.GoString(C.GooseSubscriber_getGoCbRef(subscriber)),
		StNum:     uint(C.GooseSubscriber_getStNum(subscriber)),
		SqNum:     uint(C.GooseSubscriber_getSqNum(subscriber)),
		Timestamp: time.Unix(int64(timestamp/1000), 0),
	}

	// 数据长度大小 -- 开启observer的时候才能感知到数据长度的变化，不然数据长度就是固定的。
	count := int(C.MmsValue_getArraySize(values))
	data.Entries = make([]GooseDataEntry, 0, count)

	// 数据
	for i := 0; i < count; i++ {
		// 获取数据
		cMmsValue := C.MmsValue_getElement(values, C.int(i))
		// 获取数据类型
		mmsType := MmsType(C.MmsValue_getType(cMmsValue))

		// 转换成go类型
		mmsVal, err := toGoValue(cMmsValue, mmsType)
		if err != nil {
			continue
		}
		data.Entries = append(data.Entries, GooseDataEntry{
			Index: i,
			Value: MmsValue{
				Value: mmsVal,
				Type:  mmsType,
			},
		})
		fmt.Printf("value: %v, type: %d \n", mmsVal, mmsType)
	}

	select {
	case manager.dataChan <- data:
	default:
		log.Println("Goose channel full, dropping data")
	}

}
