package iec61850

// #include <iec61850_client.h>
// #include <iec61850_common.h>
// #include "client_reporting.h"
import "C"
import (
	"fmt"
	"log"
	"sync"
	"time"
	"unsafe"
)

var (
	nextManagerID    uintptr
	reportManagersMu sync.RWMutex
	reportManagers   = make(map[uintptr]*ReportManager)
)

type ReportManager struct {
	C          *Client // go语言客户端
	RCBRef     string
	DataSetRef string
	dataChan   chan ReportData            // 数据channel
	rcb        C.ClientReportControlBlock // rcb
	dataSetDir C.LinkedList               // 数据集目录
	id         uintptr                    // ReportManager的id

	// settings   *ClientReportControlBlock  // 控制块配置
}

// 报告数据结构
type ReportData struct {
	RCBReference string
	ReportID     string
	Timestamp    time.Time
	Entries      []DataEntry
}

// 订阅的数据
type DataEntry struct {
	Name   string
	Value  MmsValue
	Reason int
}

func NewReportManager(client *Client, rcbRef string, datasetRef string, channelSize int) (*ReportManager, error) {
	if channelSize <= 0 {
		channelSize = 100
	}
	return &ReportManager{
		C:          client,
		RCBRef:     rcbRef,
		DataSetRef: datasetRef,
		dataChan:   make(chan ReportData, channelSize),
	}, nil
}

func (m *ReportManager) Subscribe() {
	var errCode C.IedClientError

	// 获取数据集目录
	dsRef := C.CString(m.DataSetRef) // simpleIOGenericIO/LLN0.Events
	defer C.free(unsafe.Pointer(dsRef))
	dataSetDir := C.IedConnection_getDataSetDirectory(m.C.conn, &errCode, dsRef, nil)
	if errCode != C.IED_ERROR_OK {
		fmt.Println("Failed to get dataset directory")
		return
	}
	// defer C.LinkedList_destroy(dataSetDir)
	m.dataSetDir = dataSetDir // 移除defer

	// 获取并配置RCB
	rcbRef := C.CString(m.RCBRef) // simpleIOGenericIO/LLN0.RP.EventsRCB01
	defer C.free(unsafe.Pointer(rcbRef))
	rcb := C.IedConnection_getRCBValues(m.C.conn, &errCode, rcbRef, nil)
	if errCode != C.IED_ERROR_OK {
		fmt.Println("Failed to get RCB values")
		return
	}
	m.rcb = rcb
	// defer C.ClientReportControlBlock_destroy(rcb)

	// 启动报告功能 -- 可以抽取出来使用client_rcb来操作
	C.ClientReportControlBlock_setRptEna(m.rcb, true)

	// 在全局注册表中注册管理器
	reportManagersMu.Lock()
	m.id = nextManagerID
	nextManagerID++
	reportManagers[m.id] = m
	reportManagersMu.Unlock()

	// 注册报告回调的回调函数
	rptId := C.ClientReportControlBlock_getRptId(rcb) // 带编号的
	C.IedConnection_installReportHandler(m.C.conn, rcbRef, rptId,
		C.ReportCallback(C.reportCallbackFunction),
		unsafe.Pointer(m.id))

	// 应用RCB设置 -- 可以抽取出来使用client_rcb来操作
	flags := C.uint32_t(C.RCB_ELEMENT_RESV) |
		C.uint32_t(C.RCB_ELEMENT_DATSET) |
		C.uint32_t(C.RCB_ELEMENT_TRG_OPS) |
		C.uint32_t(C.RCB_ELEMENT_RPT_ENA) |
		C.uint32_t(C.RCB_ELEMENT_GI)
	C.IedConnection_setRCBValues(m.C.conn, &errCode, rcb, flags, true)
	if errCode != C.IED_ERROR_OK {
		fmt.Println("Failed to set RCB values")
		return
	}
}

// 取消订阅
func (m *ReportManager) UnSubscribe() {
	var errCode C.IedClientError

	cObjectRef := C.CString(m.RCBRef)
	defer C.free(unsafe.Pointer(cObjectRef))
	C.IedConnection_uninstallReportHandler(m.C.conn, cObjectRef)

	// 停止报告
	C.ClientReportControlBlock_setRptEna(m.rcb, false)
	C.IedConnection_setRCBValues(m.C.conn, &errCode, m.rcb, C.RCB_ELEMENT_RPT_ENA, true)

	// 清理资源
	defer C.ClientReportControlBlock_destroy(m.rcb)
	defer C.LinkedList_destroy(m.dataSetDir)

	// 关闭连接
	C.IedConnection_close(m.C.conn)

	// 从全局注册表中移除
	if m.id != 0 {
		reportManagersMu.Lock()
		delete(reportManagers, m.id)
		reportManagersMu.Unlock()
		m.id = 0
	}

	// 关闭chan
	close(m.dataChan)

}

func (m *ReportManager) GetDataChan() <-chan ReportData {
	return m.dataChan
}

// Trigger手动触发报告
func (m *ReportManager) TriggerReport() {
	var errCode C.IedClientError
	cRcbRef := C.CString(m.RCBRef)
	defer C.free(unsafe.Pointer(cRcbRef))
	C.IedConnection_triggerGIReport(m.C.conn, &errCode, cRcbRef)
	if err := GetIedClientError(errCode); err != nil {
		log.Printf("触发报告失败.. %s \n", err.Error())
	} else {
		log.Printf("触发报告成功.. rbcRef: %s", m.RCBRef)
	}
}

//export goReportCallback
func goReportCallback(parameter unsafe.Pointer, report C.ClientReport) {
	// 从参数中获取ID
	id := uintptr(parameter)

	// 从全局注册表获取管理器
	reportManagersMu.Lock()
	manager := reportManagers[id]
	reportManagersMu.Unlock()

	data := ReportData{
		RCBReference: C.GoString(C.ClientReport_getRcbReference(report)),
		ReportID:     C.GoString(C.ClientReport_getRptId(report)),
	}

	// 处理时间戳
	if bool(C.ClientReport_hasTimestamp(report)) {
		unixTime := C.ClientReport_getTimestamp(report) / 1000
		data.Timestamp = time.Unix(int64(unixTime), 0)
	}

	// 处理数据集条目
	dataSetValues := C.ClientReport_getDataSetValues(report)
	dataSetDir := manager.dataSetDir

	if dataSetDir != nil {
		size := int(C.LinkedList_size(dataSetDir))
		data.Entries = make([]DataEntry, 0, size)

		for i := 0; i < size; i++ {
			reason := C.ClientReport_getReasonForInclusion(report, C.int(i))
			if reason == C.IEC61850_REASON_NOT_INCLUDED {
				continue
			}

			// 获取数据
			cMmsValue := C.MmsValue_getElement(dataSetValues, C.int(i))
			// 获取数据类型
			mmsType := MmsType(C.MmsValue_getType(cMmsValue))

			// 转换成go类型
			mmsVal, err := toGoValue(cMmsValue, mmsType)
			if err != nil {
				continue
			}

			entry := C.LinkedList_get(dataSetDir, C.int(i))
			entryName := C.GoString((*C.char)(entry.data))

			data.Entries = append(data.Entries, DataEntry{
				Name: entryName,
				Value: MmsValue{
					Value: mmsVal,
					Type:  mmsType,
				},
				Reason: int(reason),
			})
		}
	}

	select {
	case manager.dataChan <- data:
	default:
		log.Println("Report channel full, dropping data")
	}
}
