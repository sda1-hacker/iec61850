#include <iec61850_client.h>
#include <stdlib.h>
#include <stdio.h>

// 明确定义回调函数类型
typedef void (*ReportCallback)(void* parameter, ClientReport report);

// Go函数声明 -- 这个函数需要通过go语言实现
void goReportCallback(void* parameter, ClientReport report);

// C回调函数，将调用转发给Go -- 桥接函数
void reportCallbackFunction(void* parameter, ClientReport report);