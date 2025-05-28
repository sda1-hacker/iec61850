#include "goose_receiver.h"
#include "goose_subscriber.h"
#include "hal_thread.h"
#include "linked_list.h"

#include <stdlib.h>
#include <stdio.h>

// 明确定义回调函数类型
typedef void (*GooseCallback)(GooseSubscriber subscriber, void* parameter);

// Go函数声明 -- 这个函数需要通过go语言实现
void goGooseCallback(GooseSubscriber subscriber, void* parameter);

// C回调函数，将调用转发给Go -- 桥接函数
void gooseCallbackFunction(GooseSubscriber subscriber, void* parameter);