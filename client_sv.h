#include "sv_publisher.h"
#include "sv_subscriber.h"
#include "hal_thread.h"
#include "linked_list.h"

#include <stdlib.h>
#include <stdio.h>

// 明确定义回调函数类型
typedef void (*SvCallback)(SVSubscriber subscriber, void* parameter, SVSubscriber_ASDU asdu);

// Go函数声明 -- 这个函数需要通过go语言实现
void goSvCallback(SVSubscriber subscriber, void* parameter, SVSubscriber_ASDU asdu);

// C回调函数，将调用转发给Go -- 桥接函数
void svCallbackFunction(SVSubscriber subscriber, void* parameter, SVSubscriber_ASDU asdu);