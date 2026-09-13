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

package taskSocket

import (
	"testing"
	"time"

	"github.com/buexplain/netsvr-business-go/v3/contract"
)

// TestTaskSocketPoolUnavailableGateway 网关不可用时的各种失败路径
func TestTaskSocketPoolUnavailableGateway(t *testing.T) {
	// 127.0.0.1:1 上不会有服务监听
	unavailableAddr := "127.0.0.1:1"
	factory := NewFactory(unavailableAddr, time.Millisecond*300, time.Millisecond*300, time.Millisecond*300)
	pool := NewPool(2, factory, time.Millisecond*300, time.Second, []byte("~6YOt5rW35piO~"))
	defer pool.Close()
	// 取不到连接
	if socket := pool.Get(); socket != nil {
		socket.Release()
		t.Fatalf("网关不可用时不应取到连接")
	}
	manger := NewManger()
	manger.AddSocket(pool)
	if manager := manger.Count(); manager != 1 {
		t.Fatalf("注册的连接池数量：期望 1，实际 %d", manager)
	}
	// 从连接池取连接失败时，GetSockets 整体返回 nil
	if sockets := manger.GetSockets(); sockets != nil {
		t.Fatalf("网关不可用时 GetSockets 应返回 nil：%v", sockets)
	}
	// 未注册过的网关地址返回 nil
	if socket := manger.GetSocket(contract.AddrConvertToHex("127.0.0.1:2")); socket != nil {
		socket.Release()
		t.Fatalf("未注册的网关不应返回连接")
	}
	// 已注册但网关不可用时同样返回 nil
	if socket := manger.GetSocket(contract.AddrConvertToHex(unavailableAddr)); socket != nil {
		socket.Release()
		t.Fatalf("网关不可用时不应返回连接")
	}
	manger.Close()
}
