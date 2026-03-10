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
	"github.com/buexplain/netsvr-business-go/v2/contract"
	"testing"
)

func TestMainSocketManager_NewManager(t *testing.T) {
	tmp := NewManager()
	if tmp.pool == nil {
		t.Error("NewManager error")
		return
	}
}

func TestMainSocketManager_AddSocket(t *testing.T) {
	tmp := NewManager()
	mainSocket, _, _, _ := makeMainSocket()
	tmp.AddSocket(mainSocket)
	key := contract.AddrConvertToHex(mainSocket.GetAddr())
	if _, ok := tmp.pool[key]; ok == false {
		t.Error("AddSocket error")
	}
}

func TestMainSocketManager_Start_Close(t *testing.T) {
	tmp := NewManager()
	mainSocket, _, _, _ := makeMainSocket()
	tmp.AddSocket(mainSocket)
	if tmp.Start() == false {
		t.Error("Start error")
		return
	}
	if mainSocket.socket.IsConnected() == false {
		t.Error("Start error")
		return
	}
	tmp.Close()
	if tmp.connected.Load() == true {
		t.Error("Close error")
		return
	}
	if mainSocket.socket.IsConnected() == true {
		t.Error("Close error")
		return
	}
}
