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
	"bytes"
	"context"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"github.com/gorilla/websocket"
	"testing"
	"time"
)

type eventForNetBusTest struct {
}

func (e *eventForNetBusTest) onOpen(connOpen *netsvrProtocol.ConnOpen) {
	netBus.SingleCast(connOpen.UniqId, []byte(connOpen.UniqId))
}

func (e *eventForNetBusTest) onMessage(_ *netsvrProtocol.Transfer) {
}

func (e *eventForNetBusTest) onClose(_ *netsvrProtocol.ConnClose) {
}

func makeNetBus() *NetBus {
	workerAddr := "127.0.0.1:6061"
	workerHeartbeatMessage := []byte("~6YOt5rW35piO~")
	socket := NewSocket(workerAddr, time.Second*25, time.Second*25)
	events := netsvrProtocol.Event_OnOpen | netsvrProtocol.Event_OnClose | netsvrProtocol.Event_OnMessage
	mainSocket := NewMainSocket(new(eventForNetBusTest), socket, workerHeartbeatMessage, events, 10, time.Second*25)
	mainSocketManager := NewMainSocketManager()
	mainSocketManager.AddSocket(mainSocket)
	mainSocketManager.Start()
	factory := NewTaskSocketFactory(workerAddr, time.Second*10, time.Second*10)
	pool := NewTaskSocketPool(10, factory, time.Second*10, time.Second*10, workerHeartbeatMessage)
	pool.LoopHeartbeat()
	taskSocketPoolManger := NewTaskSocketPoolManger()
	taskSocketPoolManger.AddSocket(pool)
	return NewNetBus(mainSocketManager, taskSocketPoolManger)
}

type wssForNetBusTest struct {
	conn   *websocket.Conn
	uniqId string
}

func (w *wssForNetBusTest) readMessage() []byte {
	_, p, _ := w.conn.ReadMessage()
	return p
}

func (w *wssForNetBusTest) close() {
	_ = w.conn.Close()
}

func getWss() (*wssForNetBusTest, error) {
	conn, _, err := websocket.DefaultDialer.DialContext(context.Background(), "ws://127.0.0.1:6060/netsvr", nil)
	if err != nil {
		return nil, err
	}
	if _, p, err := conn.ReadMessage(); err != nil {
		_ = conn.Close()
		return nil, err
	} else {
		return &wssForNetBusTest{
			conn:   conn,
			uniqId: string(p),
		}, nil
	}
}

var netBus *NetBus

func init() {
	netBus = makeNetBus()
}

func TestNetBus_NewNetBus(t *testing.T) {
	netBus = makeNetBus()
	if netBus == nil {
		t.Error("netBus is nil")
	}
	if netBus.mainSocketManager == nil {
		t.Error("netBus.mainSocketManager is nil")
	}
	if netBus.taskSocketPoolManger == nil {
		t.Error("netBus.taskSocketPoolManger is nil")
	}
}

func TestNetBus_SingleCast(t *testing.T) {
	wss, err := getWss()
	if err != nil {
		t.Fatal("websocket dial error", "error", err)
	}
	defer wss.close()
	if wss.uniqId == "" {
		t.Error("websocket uniqId is empty")
	}
}

func TestNetBus_ConnInfo(t *testing.T) {
	wss, err := getWss()
	if err != nil {
		t.Fatal("websocket dial error", "error", err)
	}
	defer wss.close()
	ret := netBus.ConnInfo([]string{wss.uniqId}, true, true, true)
	if ret == nil {
		t.Fatal("netBus.ConnInfo return nil")
	}
	if _, ok := ret[wss.uniqId]; !ok {
		t.Error("netBus.ConnInfo return nil")
	}
}

func TestNetBus_ConnInfoUpdate(t *testing.T) {
	wss, err := getWss()
	if err != nil {
		t.Fatal("websocket dial error", "error", err)
	}
	defer wss.close()
	connInfoUpdate := &netsvrProtocol.ConnInfoUpdate{}
	connInfoUpdate.UniqId = wss.uniqId
	connInfoUpdate.NewSession = "testSession"
	connInfoUpdate.NewCustomerId = "testCustomerId"
	connInfoUpdate.NewTopics = []string{"testTopic"}
	connInfoUpdate.Data = []byte("testData")
	netBus.ConnInfoUpdate(connInfoUpdate)
	if bytes.Equal(wss.readMessage(), connInfoUpdate.Data) == false {
		t.Error("netBus.ConnInfoUpdate return nil")
	}
	connInfo := netBus.ConnInfo([]string{wss.uniqId}, true, true, true)
	if connInfo == nil {
		t.Fatal("netBus.ConnInfo return nil")
	}
	if _, ok := connInfo[wss.uniqId]; !ok {
		t.Error("netBus.ConnInfo return nil")
	}
	if connInfo[wss.uniqId].Session != connInfoUpdate.NewSession {
		t.Error("netBus.ConnInfo return nil")
	}
	if connInfo[wss.uniqId].CustomerId != connInfoUpdate.NewCustomerId {
		t.Error("netBus.ConnInfo return nil")
	}
	if connInfo[wss.uniqId].Topics[0] != connInfoUpdate.NewTopics[0] {
		t.Error("netBus.ConnInfo return nil")
	}
}
