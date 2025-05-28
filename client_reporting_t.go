package iec61850

// #include <iec61850_client.h>
// #include <iec61850_common.h>
// #include "client_reporting.h"
import "C"
import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"
	"unsafe"
)

// 订阅报告的配置
type SubscriptionManager struct {
	client     *Client
	config     SubscriptionConfig
	dataChan   chan ReportData
	errorChan  chan error
	rcb        *C.ClientReportControlBlock
	mutex      sync.Mutex
	subscribed bool
}

// 订阅配置
type SubscriptionConfig struct {
	RCBRef        string
	RCBName       string
	DataSetRef    string
	TriggerOps    TrgOps
	OptFields     OptFlds
	ReportHandler func(ReportData)
	BufferSize    int
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
	Value  string
	Reason int
}

// 创建订阅管理器
func NewSubscriptionManager(client *Client, config SubscriptionConfig) *SubscriptionManager {
	if config.BufferSize <= 0 {
		config.BufferSize = 100
	}
	return &SubscriptionManager{
		client:    client,
		config:    config,
		dataChan:  make(chan ReportData, config.BufferSize),
		errorChan: make(chan error, 10),
	}
}

// 启动订阅
func (m *SubscriptionManager) Subscribe() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return nil
}

// 停止订阅
func (m *SubscriptionManager) Unsubscribe() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.subscribed {
		return fmt.Errorf("not subscribed")
	}

	// 禁用RCB
	disableCfg := ClientReportControlBlock{Ena: false}
	if err := m.client.SetRCBValues(m.config.RCBRef, disableCfg); err != nil {
		return fmt.Errorf("disable RCB failed: %v", err)
	}

	m.subscribed = false

	// 清理资源
	close(m.dataChan)
	close(m.errorChan)
	return nil
}

// 监控连接状态
func (m *SubscriptionManager) monitorConnection() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			m.Stop()
		case <-ticker.C:
			if state := C.IedConnection_getState(m.client.conn); state != C.IED_STATE_CONNECTED {
				m.errorChan <- fmt.Errorf("connection lost")
				m.Stop()
			}
		}
	}
}

// 处理数据
func (m *SubscriptionManager) handleData() {
	for data := range m.dataChan {
		if m.config.ReportHandler != nil {
			go m.config.ReportHandler(data)
		}
	}
}

// 强制停止
func (m *SubscriptionManager) Stop() {
	m.client.Close()
}

// 数据通道
func (m *SubscriptionManager) Data() <-chan ReportData {
	return m.dataChan
}

// 错误通道
func (m *SubscriptionManager) Errors() <-chan error {
	return m.errorChan
}

//export goReportHandler
func goReportHandler(parameter unsafe.Pointer, report C.ClientReport) {
	manager := (*SubscriptionManager)(parameter)

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
	dataSetDir := (C.LinkedList)(parameter)

	if dataSetDir != nil {
		size := int(C.LinkedList_size(dataSetDir))
		data.Entries = make([]DataEntry, 0, size)

		for i := 0; i < size; i++ {
			reason := C.ClientReport_getReasonForInclusion(report, C.int(i))
			if reason == C.IEC61850_REASON_NOT_INCLUDED {
				continue
			}

			var valBuffer [500]C.char
			C.strcpy(&valBuffer[0], C.CString("no value"))

			if dataSetValues != nil {
				value := C.MmsValue_getElement(dataSetValues, C.int(i))
				if value != nil {
					C.MmsValue_printToBuffer(value, &valBuffer[0], 500)
				}
			}

			entry := C.LinkedList_get(dataSetDir, C.int(i))
			entryName := C.GoString((*C.char)(entry.data))

			data.Entries = append(data.Entries, DataEntry{
				Name:   entryName,
				Value:  C.GoString(&valBuffer[0]),
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
