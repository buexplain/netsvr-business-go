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

package mainSocket

import (
	"github.com/buexplain/netsvr-business-go/v3/contract"
	"github.com/buexplain/netsvr-business-go/v3/log"
	"github.com/buexplain/netsvr-business-go/v3/socket"
	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
	"google.golang.org/protobuf/encoding/protojson"
	"net"
	"testing"
	"time"
)

// gatewayWorkerAddr 测试环境里网关的 worker 服务地址
const gatewayWorkerAddr = "127.0.0.1:6071"

// skipUnlessGatewayReady 集成测试依赖正在运行的网关，连不上时跳过而不是失败
func skipUnlessGatewayReady(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", gatewayWorkerAddr, 500*time.Millisecond)
	if err != nil {
		t.Skipf("网关未运行（%s 不可达：%v），跳过集成测试", gatewayWorkerAddr, err)
	}
	_ = conn.Close()
}

type eventForMainSocketTest struct {
}

func (e *eventForMainSocketTest) OnOpen(connOpen *netsvrProtocol.ConnOpen) {
	log.Info("OnOpen", "connOpen", protojson.Format(connOpen))
}

func (e *eventForMainSocketTest) OnMessage(transfer *netsvrProtocol.Transfer) {
	log.Info("OnMessage", "transfer", protojson.Format(transfer))
}

func (e *eventForMainSocketTest) OnClose(connClose *netsvrProtocol.ConnClose) {
	log.Info("OnClose", "connClose", protojson.Format(connClose))
}

func makeMainSocket() (*MainSocket, contract.EventInterface, netsvrProtocol.Event, *socket.Socket) {
	h := new(eventForMainSocketTest)
	sk := socket.New(gatewayWorkerAddr, time.Second*25, time.Second*25, time.Second*25)
	events := netsvrProtocol.Event_OnOpen | netsvrProtocol.Event_OnClose | netsvrProtocol.Event_OnMessage
	mainSocket := New(h, sk, []byte("~6YOt5rW35piO~"), events, time.Second*25)
	return mainSocket, h, events, sk
}

func TestMainSocket_NewMainSocket(t *testing.T) {
	mainSocket, h, events, sk := makeMainSocket()
	if mainSocket == nil {
		t.Error("mainSocket is nil")
		return
	}
	if mainSocket.eventHandler != h {
		t.Error("eventHandler is not equal")
	}
	if mainSocket.socket != sk {
		t.Error("socket is not equal")
	}
	if mainSocket.heartbeatMessage == nil {
		t.Error("heartbeatMessage is nil")
	}
	if mainSocket.events != events {
		t.Error("events is not equal")
	}
}

func TestMainSocket_GetAddr(t *testing.T) {
	mainSocket, _, _, _ := makeMainSocket()
	if mainSocket.GetAddr() != gatewayWorkerAddr {
		t.Error("GetAddr is not equal")
	}
}

func TestMainSocket_Connect(t *testing.T) {
	skipUnlessGatewayReady(t)
	mainSocket, _, _, _ := makeMainSocket()
	if mainSocket.Connect() == false {
		t.Error("Connect failed")
		return
	}
	defer mainSocket.Close()
	if mainSocket.socket.IsConnected() == false {
		t.Error("socket is not connected")
	}
}

func TestMainSocket_Register_Unregister(t *testing.T) {
	skipUnlessGatewayReady(t)
	mainSocket, _, _, _ := makeMainSocket()
	if mainSocket.Connect() == false {
		t.Error("Connect failed")
		return
	}
	defer mainSocket.Close()
	if mainSocket.socket.IsConnected() == false {
		t.Error("socket is not connected")
		return
	}
	if mainSocket.Register() == false {
		t.Error("Register failed")
		return
	}
	mainSocket.LoopHeartbeat()
	mainSocket.LoopReceive()
	if mainSocket.Unregister() == false {
		t.Error("Unregister failed")
	}
}
