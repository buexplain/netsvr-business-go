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

package socket

import (
	"encoding/binary"
	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
	"google.golang.org/protobuf/proto"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// gatewayTaskAddr 测试环境里网关的 task 服务地址
const gatewayTaskAddr = "127.0.0.1:6072"

// skipUnlessGatewayReady 集成测试依赖正在运行的网关，连不上时跳过而不是失败
func skipUnlessGatewayReady(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", gatewayTaskAddr, 500*time.Millisecond)
	if err != nil {
		t.Skipf("网关未运行（%s 不可达：%v），跳过集成测试", gatewayTaskAddr, err)
	}
	_ = conn.Close()
}

func TestSocket_NewSocket(t *testing.T) {
	s := New(gatewayTaskAddr, time.Second*5, time.Second*5, time.Second*5)
	if s.socket != nil {
		t.Error("连接已经打开")
	}
	if s.addr != gatewayTaskAddr {
		t.Error("地址不正确")
	}
	if s.connectTimeout != time.Second*5 {
		t.Error("连接超时时间不正确")
	}
	if s.receiveTimeout != time.Second*5 || s.sendTimeout != time.Second*5 {
		t.Error("发送接收超时时间不正确")
	}
}

func TestSocket_Connect(t *testing.T) {
	skipUnlessGatewayReady(t)
	s := New(gatewayTaskAddr, time.Second*5, time.Second*5, time.Second*5)
	if s.Connect() != true {
		t.Error("连接失败")
	}
	defer s.Close()
	if s.socketBufIO == nil {
		t.Error("Socket.socketBufIO为空")
	}
	if s.IsConnected() == false {
		t.Error("连接状态不正确")
	}
}

func TestSocket_Send(t *testing.T) {
	skipUnlessGatewayReady(t)
	s := New(gatewayTaskAddr, time.Second*5, time.Second*5, time.Second*5)
	s.Connect()
	defer s.Close()
	if s.Send([]byte("~6YOt5rW35piO~")) != true {
		t.Error("发送失败")
	}
}

func TestSocket_Receive(t *testing.T) {
	skipUnlessGatewayReady(t)
	s := New(gatewayTaskAddr, time.Second*5, time.Second*5, time.Second*5)
	s.Connect()
	defer s.Close()
	message := make([]byte, 4)
	binary.BigEndian.PutUint32(message[0:4], uint32(netsvrProtocol.Cmd_TopicCount))
	s.Send(message)
	data := s.Receive()
	if data == nil {
		t.Error("接收失败")
	}
	topicCount := netsvrProtocol.TopicCountResp{}
	err := proto.Unmarshal(data[4:], &topicCount)
	if err != nil {
		t.Error("解析失败", "error", err)
	}
	s.Close()
}

func TestSocket_Close(t *testing.T) {
	skipUnlessGatewayReady(t)
	s := New(gatewayTaskAddr, time.Second*5, time.Second*5, time.Second*5)
	s.Connect()
	s.Close()
	if s.IsConnected() == true {
		t.Error("连接没有关闭")
	}
	if atomic.LoadInt32(&s.connected) != socketConnectedNo {
		t.Error("关闭失败")
	}
}

func TestSocket_ConnectFail(t *testing.T) {
	s := New("127.0.0.1:1", time.Millisecond*300, time.Millisecond*300, time.Millisecond*300)
	if s.Connect() {
		s.Close()
		t.Error("连接不存在的地址应该失败")
	}
	if s.IsConnected() {
		t.Error("连接失败后不应处于已连接状态")
	}
}

func TestSocket_SendReceiveAfterClose(t *testing.T) {
	skipUnlessGatewayReady(t)
	s := New(gatewayTaskAddr, time.Second*5, time.Second*5, time.Second*5)
	if !s.Connect() {
		t.Fatalf("连接网关失败")
	}
	if s.GetAddr() != gatewayTaskAddr {
		t.Errorf("GetAddr 不正确：%s", s.GetAddr())
	}
	s.Close()
	if s.Send([]byte("~6YOt5rW35piO~")) {
		t.Error("连接关闭后发送应该失败")
	}
	if s.Receive() != nil {
		t.Error("连接关闭后接收应该返回 nil")
	}
}
