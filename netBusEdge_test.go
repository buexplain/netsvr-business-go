/**
* Copyright 2024 buexplain@qq.com
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package netsvrBusiness

import (
	"testing"

	"github.com/buexplain/netsvr-business-go/v3/taskSocket"
)

// TestNetBusEmptyInput 发送类方法的入参为空时不应发起任何请求
func TestNetBusEmptyInput(t *testing.T) {
	bus := NewNetBus(taskSocket.NewManger())
	t.Cleanup(bus.Close)
	bus.BroadcastBulk(nil)
	bus.SingleCastBulk(nil)
	bus.SingleCastBulkByCustomerId(nil)
	bus.TopicPublishBulk(nil)
	bus.ForceOffline(nil, nil)
	bus.ForceOfflineByCustomerId(nil, nil)
	bus.ForceOfflineGuest(nil, nil, 0)
}

// TestNewNetBusPanic 连接池管理器为空时创建 NetBus 会 panic
func TestNewNetBusPanic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("NewNetBus(nil) 应该 panic")
		}
	}()
	NewNetBus(nil)
}

// TestNetBusNoGateway 没有注册任何网关、或目标不属于已注册网关时，查询返回空结果
func TestNetBusNoGateway(t *testing.T) {
	bus := NewNetBus(taskSocket.NewManger())
	t.Cleanup(bus.Close)
	// uniqId 的前 12 个十六进制字符是网关地址，这里用一个没有注册过的地址
	unknownUniqId := "0000000000000000"
	if ret := bus.CheckOnline([]string{unknownUniqId}); len(ret.UniqIds()) != 0 {
		t.Fatalf("未知网关的连接不应在线：%v", ret.UniqIds())
	}
	if ret := bus.ConnInfo([]string{unknownUniqId}, true, true, true); len(ret.Data) != 0 {
		t.Fatalf("未知网关不应返回连接信息：%v", ret.Data)
	}
	if ret := bus.ConnInfoByCustomerId([]string{"c1"}, true, true, true); ret == nil || len(ret.Data) != 0 {
		t.Fatalf("没有注册网关时不应返回连接信息：%v", ret)
	}
	if ret := bus.Limit(nil, ""); ret == nil || len(ret.Data) != 0 {
		t.Fatalf("没有注册网关时不应返回限流配置：%v", ret)
	}
	if ret := bus.Metrics(); len(ret.Data) != 0 {
		t.Fatalf("没有注册网关时不应返回统计信息：%v", ret.Data)
	}
}
