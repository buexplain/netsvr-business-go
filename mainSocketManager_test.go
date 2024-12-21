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
)

func TestMainSocketManager_NewMainSocketManager(t *testing.T) {
	tmp := NewMainSocketManager()
	if tmp.pool == nil {
		t.Error("NewMainSocketManager error")
	}
	if tmp.connected == nil {
		t.Error("NewMainSocketManager error")
	}
}

func TestMainSocketManager_GetSockets(t *testing.T) {
	tmp := NewMainSocketManager()
	if tmp.GetSockets() != nil {
		t.Error("GetSockets error")
	}
}

func TestMainSocketManager_GetSocket(t *testing.T) {
	tmp := NewMainSocketManager()
	if tmp.GetSocket("") != nil {
		t.Error("GetSocket error")
	}
}

func TestMainSocketManager_AddSocket(t *testing.T) {
	tmp := NewMainSocketManager()
	mainSocket, _, _, _ := makeMainSocket()
	tmp.AddSocket(mainSocket)
	key := WorkerAddrConvertToHex(mainSocket.GetWorkerAddr())
	if _, ok := tmp.pool[key]; ok == false {
		t.Error("AddSocket error")
	}
}

func TestMainSocketManager_Start_Close(t *testing.T) {
	tmp := NewMainSocketManager()
	mainSocket, _, _, _ := makeMainSocket()
	tmp.AddSocket(mainSocket)
	if tmp.Start() == false {
		t.Error("Start error")
	}
	if tmp.GetSockets() == nil {
		t.Error("Start error")
	}
	if mainSocket.isConnected() == false {
		t.Error("Start error")
	}
	if mainSocket.socket.IsConnected() == false {
		t.Error("Start error")
	}
	if tmp.GetSocket(WorkerAddrConvertToHex(mainSocket.GetWorkerAddr())) == nil {
		t.Error("Start error")
	}
	tmp.Close()
	if tmp.GetSockets() != nil {
		t.Error("Close error")
	}
	if tmp.GetSocket(mainSocket.GetWorkerAddr()) != nil {
		t.Error("Close error")
	}
	if tmp.connected.Load() == true {
		t.Error("Close error")
	}
	if mainSocket.isConnected() == true {
		t.Error("Close error")
	}
	if mainSocket.socket.IsConnected() == true {
		t.Error("Close error")
	}
}
