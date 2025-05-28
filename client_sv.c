#include "client_sv.h"

// 桥接函数
void svCallbackFunction(SVSubscriber subscriber, void* parameter, SVSubscriber_ASDU asdu) {
    goSvCallback(subscriber, parameter, asdu);
}