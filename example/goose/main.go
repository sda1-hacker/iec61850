package main

import (
	"fmt"
	"time"

	"github.com/sda1-hacker/iec61850"
)

func main() {
	// 测试成功奥，linux用root运行该程序
	//iec61850.GOOSETestMain("eno1")

	manager := iec61850.NewGooseManager("eno1", "simpleIOGenericIO/LLN0$GO$gcbAnalogValues", 1000, 100)

	dataChan := manager.GetDataChan()

	go manager.Subscribe()

	timer := time.NewTimer(time.Second * 5)

	for {
		select {
		case <-timer.C:
			manager.UnSubscribe()
			fmt.Printf("取消注册了.. \n")
			return
		case data := <-dataChan:
			fmt.Printf("appId: %d \n", data.AppId)
			fmt.Printf("GoId: %s \n", data.GoId)
			fmt.Printf("GoCbRef: %s \n", data.GoCbRef)
			fmt.Printf("StNum: %d \n", data.StNum)
			fmt.Printf("SqNum: %d \n", data.SqNum)
			fmt.Printf("Timestamp: %+v \n", data.Timestamp)
			for _, entry := range data.Entries {
				fmt.Printf("Index: %d, Type: %d, Value: %v \n", entry.Index, entry.Value.Type, entry.Value.Value)
			}
			fmt.Printf("======> GOOSE <====== \n")
		}
	}

}
