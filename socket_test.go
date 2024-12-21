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
	"encoding/binary"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"google.golang.org/protobuf/proto"
	"sync/atomic"
	"testing"
	"time"
)

func TestSocket_NewSocket(t *testing.T) {
	s := NewSocket("127.0.0.1:6061", time.Second*5, time.Second*5)
	if s.socket != nil {
		t.Error("连接已经打开")
	}
	if s.workerAddr != "127.0.0.1:6061" {
		t.Error("地址不正确")
	}
	if s.connectTimeout != time.Second*5 {
		t.Error("连接超时时间不正确")
	}
	if s.sendReceiveTimeout != time.Second*5 {
		t.Error("发送接收超时时间不正确")
	}
}

func TestSocket_Connect(t *testing.T) {
	s := NewSocket("127.0.0.1:6061", time.Second*5, time.Second*5)
	if s.Connect() != true {
		t.Error("连接失败")
	}
	defer s.Close()
	if s.IsConnected() == false {
		t.Error("连接状态不正确")
	}
}

func TestSocket_Send(t *testing.T) {
	s := NewSocket("127.0.0.1:6061", time.Second*5, time.Second*5)
	s.Connect()
	defer s.Close()
	if s.Send([]byte("~6YOt5rW35piO~")) != true {
		t.Error("发送失败")
	}
}

func TestSocket_Receive(t *testing.T) {
	s := NewSocket("127.0.0.1:6061", time.Second*5, time.Second*5)
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
	s := NewSocket("127.0.0.1:6061", time.Second*5, time.Second*5)
	s.Connect()
	s.Close()
	if s.IsConnected() == true {
		t.Error("连接没有关闭")
	}
	if atomic.LoadInt32(s.connected) != socketConnectedNo {
		t.Error("关闭失败")
	}
}
