package main

import "github.com/sda1_hacker/iec61850"

func main() {

	// linux上测试成功，用root权限启动
	iec61850.SVTestMain("lo")

}
