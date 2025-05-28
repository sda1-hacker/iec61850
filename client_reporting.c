#include "client_reporting.h"

// 桥接函数
void reportCallbackFunction(void* parameter, ClientReport report) {
    goReportCallback(parameter, report);
}