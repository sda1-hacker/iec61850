package main

import (
	"fmt"
	"time"

	"github.com/sda1-hacker/iec61850"
)

func main() {

	// linux上测试成功，用root权限启动
	//iec61850.SVTestMain("lo")

	asduDataStructMap := make(map[string][]iec61850.ASDUDataStruct)
	asduDataStructMap["svpub1"] = []iec61850.ASDUDataStruct{
		iec61850.ASDUDataStruct{
			Index:    1,
			DataType: iec61850.ASDU_DATATYPE_FLOAT32,
		},
		iec61850.ASDUDataStruct{
			Index:    2,
			DataType: iec61850.ASDU_DATATYPE_FLOAT32,
		},
		iec61850.ASDUDataStruct{
			Index:    3,
			DataType: iec61850.ASDU_DATATYPE_TIMESTAMP,
		},
	}

	asduDataStructMap["svpub2"] = []iec61850.ASDUDataStruct{
		iec61850.ASDUDataStruct{
			Index:    1,
			DataType: iec61850.ASDU_DATATYPE_FLOAT32,
		},
		iec61850.ASDUDataStruct{
			Index:    2,
			DataType: iec61850.ASDU_DATATYPE_FLOAT32,
		},
		iec61850.ASDUDataStruct{
			Index:    3,
			DataType: iec61850.ASDU_DATATYPE_TIMESTAMP,
		},
	}

	manager := iec61850.NewSvManager("lo0", 0x4000, asduDataStructMap, 100)

	dataChan := manager.GetDataChan()

	// 启动一个携程监听goose事件
	go manager.Subscribe()

	timer := time.NewTimer(time.Second * 5)

	for {
		select {
		case <-timer.C:
			manager.UnSubscribe()
			fmt.Printf("取消注册了.. \n")
			return
		case data := <-dataChan:
			fmt.Printf("SvID: %d \n", data.SvID)
			fmt.Printf("ConfRev: %s \n", data.ConfRev)
			fmt.Printf("RefrTm: %s \n", data.RefrTm)
			fmt.Printf("StNum: %d \n", data.SmpCnt)
			for _, entry := range data.Entries {
				fmt.Printf("Index: %d, Type: %d, Value: %v \n", entry.Index, entry.Asdu_DataType, entry.Value)
			}
			fmt.Printf("======> SV <====== \n")
		}
	}
}
