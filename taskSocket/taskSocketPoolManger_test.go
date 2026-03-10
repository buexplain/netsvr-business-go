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
	"github.com/buexplain/netsvr-business-go/v2/contract"
	"testing"
	"time"
)

func TestTaskSocketPoolManger_NewManger(t *testing.T) {
	poolManger := NewManger()
	if poolManger.pools == nil {
		t.Error("NewManger failed")
	}
	poolManger.Close()
}

func TestTaskSocketPoolManger_AddSocket_GetSocket(t *testing.T) {
	poolManger := NewManger()
	factory := NewFactory("127.0.0.1:6062", time.Second*10, time.Second*10, time.Second*10)
	pool := NewPool(10, factory, time.Second*10, time.Second*10, []byte("~6YOt5rW35piO~"))
	pool.LoopHeartbeat()
	poolManger.AddSocket(pool)
	if poolManger.Count() != 1 {
		t.Error("AddSocket failed")
	}
	taskSocket := poolManger.GetSocket(contract.AddrConvertToHex(factory.GetAddr()))
	if taskSocket == nil {
		t.Error("GetSocket failed")
		return
	}
	taskSocket.Release()
	poolManger.Close()
}

func TestTaskSocketPoolManger_GetSockets(t *testing.T) {
	poolManger := NewManger()
	factory := NewFactory("127.0.0.1:6062", time.Second*10, time.Second*10, time.Second*10)
	pool := NewPool(10, factory, time.Second*10, time.Second*10, []byte("~6YOt5rW35piO~"))
	pool.LoopHeartbeat()
	poolManger.AddSocket(pool)
	if poolManger.Count() != 1 {
		t.Error("AddSocket failed")
	}
	taskSockets := poolManger.GetSockets()
	if taskSockets == nil {
		t.Error("GetSockets failed")
		return
	}
	for _, taskSocket := range taskSockets {
		taskSocket.Release()
	}
	poolManger.Close()
}

func TestTaskSocketPoolManger_Close(t *testing.T) {
	poolManger := NewManger()
	factory := NewFactory("127.0.0.1:6062", time.Second*10, time.Second*10, time.Second*10)
	pool := NewPool(10, factory, time.Second*10, time.Second*10, []byte("~6YOt5rW35piO~"))
	pool.LoopHeartbeat()
	poolManger.AddSocket(pool)
	if poolManger.Count() != 1 {
		t.Error("AddSocket failed")
		return
	}
	poolManger.Close()
	if poolManger.Count() != 0 {
		t.Error("Close failed")
	}
}
