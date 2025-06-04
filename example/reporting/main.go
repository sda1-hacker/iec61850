package main

import (
	"fmt"
	"github.com/sda1-hacker/iec61850"
	"log"
	"time"
)

// import "github.com/sda1-hacker/iec61850"

func main() {

	// 测试成功奥，mac本地可以直接读取数据，linux应该也没有问题
	// iec61850.ReportingMain()

	settings := iec61850.NewSettings()
	settings.Host = "127.0.0.1"
	settings.Port = 102
	client, err := iec61850.NewClient(settings)
	if err != nil {
		log.Printf("iec61850 mms客户端连接失败..")
		return
	}
	rcbRef := "simpleIOGenericIO/LLN0.RP.EventsRCB01"
	dataSetRef := "simpleIOGenericIO/LLN0.Events"
	manager, err := iec61850.NewReportManager(client, rcbRef, dataSetRef, 100)
	if err != nil {
		log.Printf("iec61850 mms客户端订阅失败..")
		return
	}
	manager.Subscribe()
	defer manager.UnSubscribe()

	ticker := time.NewTicker(time.Second * 3)

	dataCh := manager.GetDataChan()

	for {
		select {
		// 触发报告
		case <-ticker.C:
			manager.TriggerReport()
		// 获取数据
		case dta := <-dataCh:
			fmt.Printf("ReportID: %s \n", dta.ReportID)
			fmt.Printf("RCBReference: %s \n", dta.RCBReference)
			fmt.Printf("Timestamp: %d \n", dta.Timestamp.Unix())
			for _, entry := range dta.Entries {
				fmt.Printf("name: %s, reason: %d, type: %d, value: %v \n", entry.Name, entry.Reason, entry.Value.Type, entry.Value.Value)
			}
			fmt.Printf("=======> <======= \n")
		}
	}

}
