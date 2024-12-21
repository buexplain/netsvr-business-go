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
	"encoding/binary"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"log/slog"
	"strings"
	"testing"
	"time"
)

type eventExample struct {
}

func (e *eventExample) onOpen(connOpen *netsvrProtocol.ConnOpen) {
	logger.Info("onOpen", "connOpen", protojson.Format(connOpen))
}

func (e *eventExample) onMessage(transfer *netsvrProtocol.Transfer) {
	logger.Info("onMessage", "transfer", protojson.Format(transfer))
}

func (e *eventExample) onClose(connClose *netsvrProtocol.ConnClose) {
	logger.Info("onClose", "connClose", protojson.Format(connClose))
}

func makeMainSocket() (*MainSocket, EventInterface, netsvrProtocol.Event, *Socket) {
	h := new(eventExample)
	socket := NewSocket("127.0.0.1:6061", time.Second*25, time.Second*25)
	events := netsvrProtocol.Event_OnOpen | netsvrProtocol.Event_OnClose | netsvrProtocol.Event_OnMessage
	mainSocket := NewMainSocket(h, socket, []byte("~6YOt5rW35piO~"), events, 10, time.Second*25)
	return mainSocket, h, events, socket
}

func TestMainSocket_NewMainSocket(t *testing.T) {
	mainSocket, h, events, socket := makeMainSocket()
	if mainSocket == nil {
		t.Error("mainSocket is nil")
	}
	if mainSocket.eventHandler != h {
		t.Error("eventHandler is not equal")
	}
	if mainSocket.socket != socket {
		t.Error("socket is not equal")
	}
	if mainSocket.workerHeartbeatMessage == nil {
		t.Error("workerHeartbeatMessage is nil")
	}
	if mainSocket.events != events {
		t.Error("events is not equal")
	}
	if mainSocket.processCmdGoroutineNum != 10 {
		t.Error("processCmdGoroutineNum is not equal")
	}
}

func TestMainSocket_GetWorkerAddr(t *testing.T) {
	mainSocket, _, _, _ := makeMainSocket()
	if mainSocket.GetWorkerAddr() != "127.0.0.1:6061" {
		t.Error("GetWorkerAddr is not equal")
	}
}

func TestMainSocket_Connect(t *testing.T) {
	mainSocket, _, _, _ := makeMainSocket()
	if mainSocket.Connect() == false {
		t.Error("Connect failed")
	}
	defer mainSocket.Close()
	if mainSocket.isConnected() == false {
		t.Error("socket is closed")
	}
	if mainSocket.socket.IsConnected() == false {
		t.Error("socket is not connected")
	}
}

func TestMainSocket_Register_Unregister(t *testing.T) {
	mainSocket, _, _, _ := makeMainSocket()
	if mainSocket.Connect() == false {
		t.Error("Connect failed")
	}
	defer mainSocket.Close()
	if mainSocket.isConnected() == false {
		t.Error("socket is closed")
	}
	if mainSocket.socket.IsConnected() == false {
		t.Error("socket is not connected")
	}
	if mainSocket.Register() == false {
		t.Error("Register failed")
	}
	if mainSocket.Unregister() == false {
		t.Error("Unregister failed")
	}
}

func TestMainSocket_Send_Receive(t *testing.T) {
	stdOut := bytes.NewBuffer(nil)
	defaultLog := logger
	logger = slog.New(slog.NewTextHandler(stdOut, nil))
	defer func() {
		logger = defaultLog
	}()
	mainSocket, _, _, _ := makeMainSocket()
	if mainSocket.Connect() == false {
		t.Error("Connect failed")
	}
	defer mainSocket.Close()
	if mainSocket.isConnected() == false {
		t.Error("socket is closed")
	}
	if mainSocket.socket.IsConnected() == false {
		t.Error("socket is not connected")
	}
	if mainSocket.Register() == false {
		t.Error("Register failed")
	}
	mainSocket.LoopSend()
	mainSocket.LoopReceive()
	wss, _, err := websocket.DefaultDialer.DialContext(context.Background(), "ws://127.0.0.1:6060/netsvr", nil)
	if err != nil {
		t.Error("websocket dial error", "error", err)
	}
	//从客户端发送消息到服务端, 用于测试服务端是否能收到onmessage事件
	if err := wss.WriteMessage(websocket.TextMessage, []byte("hello world")); err != nil {
		t.Error("websocket write message error", "error", err)
	}
	//从服务端发送一条数据到客户端, 用于测试客户端是否能收到onmessage事件
	message := make([]byte, 4)
	binary.BigEndian.PutUint32(message[0:4], uint32(netsvrProtocol.Cmd_Broadcast))
	broadcast := netsvrProtocol.Broadcast{}
	broadcast.Data = []byte("hello world")
	message, _ = (proto.MarshalOptions{}).MarshalAppend(message, &broadcast)
	mainSocket.Send(message)
	//检查服务端的发送能力是否正常
	if _, message, err := wss.ReadMessage(); err != nil {
		t.Error("websocket read message error", "error", err)
	} else if bytes.EqualFold(message, broadcast.Data) == false {
		t.Error("websocket read message error", "error", "data not equal")
	}
	time.Sleep(time.Second * 1)
	_ = wss.Close()
	time.Sleep(time.Second * 1)
	//检查服务端的接收能力是否正常
	logStr := stdOut.String()
	if strings.Contains(logStr, "connOpen=") == false {
		t.Error("mainSocket connOpen error", "logStr", logStr)
	}
	if strings.Contains(logStr, "connClose=") == false {
		t.Error("mainSocket connClose error", "logStr", logStr)
	}
	if strings.Contains(logStr, "transfer=") == false {
		t.Error("mainSocket onMessage error", "logStr", logStr)
	}
}

func TestMainSocket_Close(t *testing.T) {
	mainSocket, _, _, _ := makeMainSocket()
	if mainSocket.Connect() == false {
		t.Error("Connect failed")
	}
	if mainSocket.isConnected() == false {
		t.Error("Connect failed")
	}
	if mainSocket.socket.IsConnected() == false {
		t.Error("Connect failed")
	}
	if mainSocket.Register() == false {
		t.Error("Register failed")
	}
	mainSocket.LoopHeartbeat()
	mainSocket.LoopSend()
	mainSocket.LoopReceive()
	mainSocket.Unregister()
	mainSocket.Close()
	if mainSocket.isConnected() == true {
		t.Error("Close failed")
	}
	if mainSocket.socket.IsConnected() == true {
		t.Error("Close failed")
	}
}
