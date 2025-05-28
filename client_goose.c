#include "client_goose.h"

// 桥接函数
void gooseCallbackFunction(GooseSubscriber subscriber, void* parameter) {
    goGooseCallback(subscriber, parameter);
}