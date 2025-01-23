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
	"time"
)

func TestTaskSocketPoolManger_NewTaskSocketPoolManger(t *testing.T) {
	poolManger := NewTaskSocketPoolManger()
	if poolManger.pools == nil {
		t.Error("NewTaskSocketPoolManger failed")
	}
	poolManger.Close()
}

func TestTaskSocketPoolManger_AddSocket_GetSocket(t *testing.T) {
	poolManger := NewTaskSocketPoolManger()
	factory := NewTaskSocketFactory("127.0.0.1:6061", time.Second*10, time.Second*10, time.Second*10)
	pool := NewTaskSocketPool(10, factory, time.Second*10, time.Second*10, []byte("~6YOt5rW35piO~"))
	pool.LoopHeartbeat()
	poolManger.AddSocket(pool)
	if poolManger.Count() != 1 {
		t.Error("AddSocket failed")
	}
	taskSocket := poolManger.GetSocket(WorkerAddrConvertToHex(factory.GetWorkerAddr()))
	if taskSocket == nil {
		t.Error("GetSocket failed")
	}
	taskSocket.Release()
	poolManger.Close()
}

func TestTaskSocketPoolManger_GetSockets(t *testing.T) {
	poolManger := NewTaskSocketPoolManger()
	factory := NewTaskSocketFactory("127.0.0.1:6061", time.Second*10, time.Second*10, time.Second*10)
	pool := NewTaskSocketPool(10, factory, time.Second*10, time.Second*10, []byte("~6YOt5rW35piO~"))
	pool.LoopHeartbeat()
	poolManger.AddSocket(pool)
	if poolManger.Count() != 1 {
		t.Error("AddSocket failed")
	}
	taskSockets := poolManger.GetSockets()
	if taskSockets == nil {
		t.Error("GetSockets failed")
	}
	for _, taskSocket := range taskSockets {
		taskSocket.Release()
	}
	poolManger.Close()
}

func TestTaskSocketPoolManger_Close(t *testing.T) {
	poolManger := NewTaskSocketPoolManger()
	factory := NewTaskSocketFactory("127.0.0.1:6061", time.Second*10, time.Second*10, time.Second*10)
	pool := NewTaskSocketPool(10, factory, time.Second*10, time.Second*10, []byte("~6YOt5rW35piO~"))
	pool.LoopHeartbeat()
	poolManger.AddSocket(pool)
	if poolManger.Count() != 1 {
		t.Error("AddSocket failed")
	}
	poolManger.Close()
	if poolManger.Count() != 0 {
		t.Error("Close failed")
	}
}
