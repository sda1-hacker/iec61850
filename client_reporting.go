package iec61850

//// #include <iec61850_client.h>
//// #include <iec61850_common.h>
//// #include "client_reporting.h"
//import "C"
//import (
//	"fmt"
//	"log"
//	"os"
//	"os/signal"
//	"strconv"
//	"sync/atomic"
//	"time"
//	"unsafe"
//)
//
//var running int32 = 1
//
////export goReportCallback
//func goReportCallback(parameter unsafe.Pointer, report C.ClientReport) {
//	dataSetDirectory := (C.LinkedList)(parameter)
//	dataSetValues := C.ClientReport_getDataSetValues(report)
//
//	rcbRef := C.GoString(C.ClientReport_getRcbReference(report))
//	rptId := C.GoString(C.ClientReport_getRptId(report))
//	log.Printf("received report for %s with rptId %s\n", rcbRef, rptId)
//
//	if bool(C.ClientReport_hasTimestamp(report)) {
//		unixTime := C.ClientReport_getTimestamp(report) / 1000
//		t := time.Unix(int64(unixTime), 0)
//		fmt.Printf("  report contains timestamp (%d): %s\n",
//			unixTime, t.Format(time.RFC3339))
//	}
//
//	if dataSetDirectory != nil {
//		size := int(C.LinkedList_size(dataSetDirectory))
//		for i := 0; i < size; i++ {
//			reason := C.ClientReport_getReasonForInclusion(report, C.int(i))
//			if reason == C.IEC61850_REASON_NOT_INCLUDED {
//				continue
//			}
//
//			var valBuffer [500]C.char
//			C.strcpy(&valBuffer[0], C.CString("no value"))
//
//			if dataSetValues != nil {
//				value := C.MmsValue_getElement(dataSetValues, C.int(i))
//				if value != nil {
//					cMmsType := C.MmsValue_getType(value)
//					log.Printf("类型: %v", cMmsType)
//					C.MmsValue_printToBuffer(value, &valBuffer[0], 500)
//				}
//			}
//
//			entry := C.LinkedList_get(dataSetDirectory, C.int(i))
//			entryName := C.GoString((*C.char)(entry.data))
//			fmt.Printf("  %s (reason %d): %s\n",
//				entryName, reason, C.GoString(&valBuffer[0]))
//		}
//	}
//
//}
//
//func ReportingMain() {
//	// server_example_basic_io or server_example_goose.
//	// 处理命令行参数
//	hostname := "localhost"
//	tcpPort := 102
//	if len(os.Args) > 1 {
//		hostname = os.Args[1]
//	}
//	if len(os.Args) > 2 {
//		if port, err := strconv.Atoi(os.Args[2]); err == nil {
//			tcpPort = port
//		}
//	}
//
//	// 创建连接
//	con := C.IedConnection_create()
//	defer C.IedConnection_destroy(con)
//
//	cHost := C.CString(hostname)
//	defer C.free(unsafe.Pointer(cHost))
//
//	var errCode C.IedClientError
//	C.IedConnection_connect(con, &errCode, cHost, C.int(tcpPort))
//	if errCode != C.IED_ERROR_OK {
//		fmt.Printf("Connection failed to %s:%d\n", hostname, tcpPort)
//		return
//	}
//
//	// 获取数据集目录
//	dsRef := C.CString("simpleIOGenericIO/LLN0.Events")
//	defer C.free(unsafe.Pointer(dsRef))
//	dataSetDir := C.IedConnection_getDataSetDirectory(con, &errCode, dsRef, nil)
//	if errCode != C.IED_ERROR_OK {
//		fmt.Println("Failed to get dataset directory")
//		return
//	}
//	defer C.LinkedList_destroy(dataSetDir)
//
//	// 读取数据集
//	//clientDS := C.IedConnection_readDataSetValues(con, &errCode, dsRef, nil)
//	//if clientDS == nil {
//	//	fmt.Println("Failed to read dataset values")
//	//}
//	//defer C.ClientDataSet_destroy(clientDS)
//
//	// 获取并配置RCB -- 带编号的rbc
//	// simpleIOGenericIO/LLN0.RP.EventsRCB01
//	rcbRef := C.CString("simpleIOGenericIO/LLN0.RP.EventsRCB01")
//	defer C.free(unsafe.Pointer(rcbRef))
//	rcb := C.IedConnection_getRCBValues(con, &errCode, rcbRef, nil)
//	if errCode != C.IED_ERROR_OK {
//		fmt.Println("Failed to get RCB values")
//		return
//	}
//	defer C.ClientReportControlBlock_destroy(rcb)
//
//	// 设置RCB参数
//	// C.ClientReportControlBlock_setResv(rcb, true) // 客户端是否独占rbc
//	// C.ClientReportControlBlock_setTrgOps(rcb,
//	// 	C.TRG_OPT_DATA_CHANGED|C.TRG_OPT_QUALITY_CHANGED|C.TRG_OPT_GI) //
//	// dsRefNew := C.CString("simpleIOGenericIO/LLN0$Events") // $控制快
//	// defer C.free(unsafe.Pointer(dsRefNew))
//	// C.ClientReportControlBlock_setDataSetReference(rcb, dsRefNew)
//	// C.ClientReportControlBlock_setGI(rcb, true) // 连接的时候触发
//	C.ClientReportControlBlock_setRptEna(rcb, true) // 启动报告功能
//
//	// 注册报告回调 // 不带编号的rbc，建议改成带编号的
//	// simpleIOGenericIO/LLN0.RP.EventsRCB
//	rcbName := C.CString("simpleIOGenericIO/LLN0.RP.EventsRCB01")
//	defer C.free(unsafe.Pointer(rcbName))
//	rptId := C.ClientReportControlBlock_getRptId(rcb)
//	C.IedConnection_installReportHandler(con, rcbName, rptId,
//		C.ReportCallback(C.reportCallbackFunction),
//		unsafe.Pointer(dataSetDir))
//
//	// 应用RCB设置
//	flags := C.uint32_t(C.RCB_ELEMENT_RESV) |
//		C.uint32_t(C.RCB_ELEMENT_DATSET) |
//		C.uint32_t(C.RCB_ELEMENT_TRG_OPS) |
//		C.uint32_t(C.RCB_ELEMENT_RPT_ENA) |
//		C.uint32_t(C.RCB_ELEMENT_GI)
//	C.IedConnection_setRCBValues(con, &errCode, rcb, flags, true)
//	if errCode != C.IED_ERROR_OK {
//		fmt.Println("Failed to set RCB values")
//		return
//	}
//
//	// 触发GI报告
//	time.Sleep(1 * time.Second)
//	//C.ClientReportControlBlock_setGI(rcb, true)
//	//C.IedConnection_setRCBValues(con, &errCode, rcb, C.RCB_ELEMENT_GI, true)
//	//if errCode != C.IED_ERROR_OK {
//	//	fmt.Println("Failed to trigger GI report")
//	//}
//
//	// 信号处理
//	sigCh := make(chan os.Signal, 1)
//	signal.Notify(sigCh, os.Interrupt)
//	go func() {
//		<-sigCh
//		atomic.StoreInt32(&running, 0)
//	}()
//
//	// 主循环
//	for atomic.LoadInt32(&running) == 1 {
//		time.Sleep(1000 * time.Millisecond)
//		if state := C.IedConnection_getState(con); state != C.IED_STATE_CONNECTED {
//			fmt.Println("Connection lost")
//			atomic.StoreInt32(&running, 0)
//		} else {
//			fmt.Println("Connected !")
//		}
//
//		// 手动触发
//		fmt.Println("周期触发报告！！")
//		C.IedConnection_triggerGIReport(con, &errCode, rcbRef)
//		if err := GetIedClientError(errCode); err != nil {
//			fmt.Println("GetIedClientError")
//		} else {
//
//		}
//
//	}
//
//	// 清理
//	C.ClientReportControlBlock_setRptEna(rcb, false)
//	C.IedConnection_setRCBValues(con, &errCode, rcb, C.RCB_ELEMENT_RPT_ENA, true)
//	C.IedConnection_close(con)
//}
