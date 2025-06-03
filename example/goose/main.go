package main

import (
	"github.com/sda1_hacker/iec61850"
)

func main() {
	// 测试成功奥，linux用root运行该程序
	iec61850.GOOSETestMain("lo0")
}
