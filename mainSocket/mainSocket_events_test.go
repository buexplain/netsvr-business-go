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
	"testing"
	"time"

	"github.com/buexplain/netsvr-business-go/v3/socket"
	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
	"github.com/gorilla/websocket"
)

// customerWsURL 测试环境里网关的 customer（websocket）服务地址
const customerWsURL = "ws://127.0.0.1:6070/netsvr"

// 事件的缓冲区大小。注册到网关后会收到所有客户连接的事件，
// 并行跑其它包的测试时也会有别的连接的事件混进来，通道留大一些避免丢弃目标事件。
const eventChanCap = 1024

// eventRecorder 记录网关转发过来的事件
type eventRecorder struct {
	opened  chan *netsvrProtocol.ConnOpen
	message chan *netsvrProtocol.Transfer
	closed  chan *netsvrProtocol.ConnClose
}

func newEventRecorder() *eventRecorder {
	return &eventRecorder{
		opened:  make(chan *netsvrProtocol.ConnOpen, eventChanCap),
		message: make(chan *netsvrProtocol.Transfer, eventChanCap),
		closed:  make(chan *netsvrProtocol.ConnClose, eventChanCap),
	}
}

func (e *eventRecorder) OnOpen(connOpen *netsvrProtocol.ConnOpen) {
	select {
	case e.opened <- connOpen:
	default:
	}
}

func (e *eventRecorder) OnMessage(transfer *netsvrProtocol.Transfer) {
	select {
	case e.message <- transfer:
	default:
	}
}

func (e *eventRecorder) OnClose(connClose *netsvrProtocol.ConnClose) {
	select {
	case e.closed <- connClose:
	default:
	}
}

// waitUniqIdEvent 从事件通道里取出指定 uniqId 的事件，忽略其它连接的事件
func waitUniqIdEvent[T any](t *testing.T, what string, events <-chan T, uniqId string, getUniqId func(T) string) T {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case event := <-events:
			if getUniqId(event) == uniqId {
				return event
			}
		case <-deadline:
			t.Fatalf("等待 %s 事件超时（uniqId=%s）", what, uniqId)
			var zero T
			return zero
		}
	}
}

// TestMainSocket_Events 注册后能收到连接打开、数据透传、连接关闭三个事件
func TestMainSocket_Events(t *testing.T) {
	skipUnlessGatewayReady(t)
	handler := newEventRecorder()
	sk := socket.New(gatewayWorkerAddr, time.Second*10, time.Second*10, time.Second*10)
	events := netsvrProtocol.Event_OnOpen | netsvrProtocol.Event_OnClose | netsvrProtocol.Event_OnMessage
	mainSocket := New(handler, sk, []byte("~6YOt5rW35piO~"), events, time.Second*10)
	if !mainSocket.Connect() {
		t.Fatalf("连接网关的 worker 服务失败")
	}
	t.Cleanup(mainSocket.Close)
	if !mainSocket.Register() {
		t.Fatalf("注册到网关失败")
	}
	mainSocket.LoopReceive()
	// 建立一个客户连接，触发 OnOpen
	conn, _, err := websocket.DefaultDialer.Dial(customerWsURL, nil)
	if err != nil {
		t.Fatalf("连接网关的 customer 服务失败：%v", err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("设置读超时失败：%v", err)
	}
	_, uniqIdMessage, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("读取网关下发的 uniqId 失败：%v", err)
	}
	uniqId := string(uniqIdMessage)
	waitUniqIdEvent(t, "OnOpen", handler.opened, uniqId, func(event *netsvrProtocol.ConnOpen) string {
		return event.GetUniqId()
	})
	// 客户发送数据，触发 OnMessage
	data := "hello-netsvr"
	if err := conn.WriteMessage(websocket.TextMessage, []byte(data)); err != nil {
		t.Fatalf("客户发送数据失败：%v", err)
	}
	transfer := waitUniqIdEvent(t, "OnMessage", handler.message, uniqId, func(event *netsvrProtocol.Transfer) string {
		return event.GetUniqId()
	})
	if string(transfer.GetData()) != data {
		t.Fatalf("OnMessage 事件的数据：期望 %q，实际 %q", data, transfer.GetData())
	}
	// 关闭客户连接，触发 OnClose
	_ = conn.Close()
	waitUniqIdEvent(t, "OnClose", handler.closed, uniqId, func(event *netsvrProtocol.ConnClose) string {
		return event.GetUniqId()
	})
}
