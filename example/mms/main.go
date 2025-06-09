package main

import (
	"log"

	"github.com/sda1-hacker/iec61850"
)

func main() {

	settings := iec61850.NewSettings()
	settings.Host = "127.0.0.1"
	settings.Port = 102

	c, err := iec61850.NewClient(settings)
	if err != nil {
		log.Printf("client conn error")
	}

	// DoRead(c, "TEMPLATEPROT/LLN0.LQ")

	DoReadStruct(c, "simpleIOGenericIO/LLN0.Events") // 用了libiec61850

}

// 读取结构体
func DoRead(c *iec61850.Client, obf string) {
	read, err := c.Read(obf, iec61850.ST)
	if err != nil {
		log.Printf("read error ")
	}
	if ab, ok := read.([]*iec61850.MmsValue); ok {
		for _, a := range ab {
			log.Printf("value ==> %v , type ==> %v", a.Value, a.Type)
		}
	}
}

// 读取结构体
func DoReadStruct(c *iec61850.Client, obf string) {
	dataset, err := c.ReadDataSet(obf)
	if err != nil {
		log.Printf("read DataSet error %v \n", err.Error())
	}
	for _, ds := range dataset {
		log.Printf("ds.Type: %v, ds.Value: %v \n", ds.Type, ds.Value)
	}
}
