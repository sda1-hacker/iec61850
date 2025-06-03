package iec61850

// #include "client_goose.h"
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"time"
	"unsafe"
)

//export goGooseCallback
func goGooseCallback(subscriber C.GooseSubscriber, parameter unsafe.Pointer) {

	// 从C结构获取数据
	stNum := uint(C.GooseSubscriber_getStNum(subscriber))
	sqNum := uint(C.GooseSubscriber_getSqNum(subscriber))
	timestamp := uint64(C.GooseSubscriber_getTimestamp(subscriber))
	valid := bool(C.GooseSubscriber_isValid(subscriber))

	// 获取数据集值
	var buf [1024]byte
	values := C.GooseSubscriber_getDataSetValues(subscriber)
	C.MmsValue_printToBuffer(values, (*C.char)(unsafe.Pointer(&buf[0])), 1024)
	data := C.GoString((*C.char)(unsafe.Pointer(&buf[0])))

	// 打印结果
	fmt.Printf("\n=== GOOSE Event ===\n")
	fmt.Printf("StNum: %d  SqNum: %d\n", stNum, sqNum)
	fmt.Printf("Timestamp: %s\n", time.Unix(int64(timestamp/1000), int64(timestamp%1000)*1e6))
	fmt.Printf("Valid: %t\n", valid)
	fmt.Printf("Data: %s\n", data)

}

// 网卡名称
func GOOSETestMain(ifaceName string) {
	// 创建接收器
	receiver := C.GooseReceiver_create()

	// 设置网络接口（默认eth0）
	iface := C.CString(ifaceName)
	defer C.free(unsafe.Pointer(iface))
	C.GooseReceiver_setInterfaceId(receiver, iface)

	// 创建订阅者
	// gocbRef := C.CString("simpleIOGenericIO/LLN0$GO$gcbAnalogValues")
	gocbRef := C.CString("")
	defer C.free(unsafe.Pointer(gocbRef))
	subscriber := C.GooseSubscriber_create(gocbRef, nil)

	// 设置目标MAC地址 (01:0c:cd:01:00:01)
	var dstMac [6]C.uint8_t = [6]C.uint8_t{0x01, 0x0c, 0xcd, 0x01, 0x00, 0x01}
	C.GooseSubscriber_setDstMac(subscriber, &dstMac[0])

	// 设置AppID
	C.GooseSubscriber_setAppId(subscriber, 1000)

	// 设置回调
	C.GooseSubscriber_setListener(subscriber, C.GooseCallback(C.gooseCallbackFunction), nil)

	fmt.Printf("监听网卡 %v \n", ifaceName)

	// 添加订阅者到接收器
	C.GooseReceiver_addSubscriber(receiver, subscriber)

	// 启动接收器
	C.GooseReceiver_start(receiver)

	// 等待中断信号
	fmt.Println("GOOSE Subscriber started (Press Ctrl+C to exit)...")
	for {
		// 简单的循环等待
		time.Sleep(1 * time.Second)
	}

	// 清理资源
	C.GooseReceiver_stop(receiver)
	C.GooseSubscriber_destroy(subscriber)
	C.GooseReceiver_destroy(receiver)
}
