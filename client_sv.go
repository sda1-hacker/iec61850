package iec61850

// #include "client_sv.h"
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

//export goSvCallback
func goSvCallback(subscriber C.SVSubscriber, parameter unsafe.Pointer, asdu C.SVSubscriber_ASDU) {
	// 获取基本参数
	svID := C.GoString(C.SVSubscriber_ASDU_getSvId(asdu))
	smpCnt := int(C.SVSubscriber_ASDU_getSmpCnt(asdu))
	dataSize := int(C.SVSubscriber_ASDU_getDataSize(asdu))

	fmt.Printf("\n=== SV Update ===\n")
	fmt.Printf("SV ID: %s\n", svID)
	fmt.Printf("Sample Count: %d\n", smpCnt)
	fmt.Printf("Data Size: %d bytes\n", dataSize)

	// 打印前两个浮点值（假设数据格式为FLOAT32）
	if dataSize >= 8 {
		val1 := float32(C.SVSubscriber_ASDU_getFLOAT32(asdu, 0))
		val2 := float32(C.SVSubscriber_ASDU_getFLOAT32(asdu, 4))
		fmt.Printf("Value 1: %.2f\n", val1)
		fmt.Printf("Value 2: %.2f\n", val2)
	}
}

func SVTestMain(ifaceName string) {
	// 创建接收器
	receiver := C.SVReceiver_create()
	defer C.SVReceiver_destroy(receiver)

	// 设置网络接口
	iface := ifaceName
	if len(os.Args) > 1 {
		iface = os.Args[1]
	}
	cIface := C.CString(iface)
	C.SVReceiver_setInterfaceId(receiver, cIface)
	defer C.free(unsafe.Pointer(cIface))

	// 创建订阅者（APPID 0x4000）
	subscriber := C.SVSubscriber_create(nil, 0x4000)
	defer C.SVSubscriber_destroy(subscriber)

	// 设置回调函数
	C.SVSubscriber_setListener(subscriber, C.SvCallback(C.svCallbackFunction), nil)

	// 添加订阅者到接收器
	C.SVReceiver_addSubscriber(receiver, subscriber)

	// 启动接收器
	C.SVReceiver_start(receiver)
	defer C.SVReceiver_stop(receiver)

	// 等待退出信号
	fmt.Println("SV Subscriber started (Ctrl+C to exit)...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
}
