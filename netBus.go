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
	"fmt"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"google.golang.org/protobuf/proto"
)

type NetBus struct {
	mainSocketManager    *MainSocketManager
	taskSocketPoolManger *TaskSocketPoolManger
}

func NewNetBus(mainSocketManager *MainSocketManager, taskSocketPoolManger *TaskSocketPoolManger) *NetBus {
	if taskSocketPoolManger == nil {
		panic("taskSocketPoolManger is nil")
	}
	return &NetBus{
		mainSocketManager:    mainSocketManager,
		taskSocketPoolManger: taskSocketPoolManger,
	}
}

func (n *NetBus) ConnInfoUpdate(connInfoUpdate *netsvrProtocol.ConnInfoUpdate) {
	message := n.pack(netsvrProtocol.Cmd_ConnInfoUpdate, connInfoUpdate)
	n.sendToSocketByUniqId(connInfoUpdate.GetUniqId(), message)
}

func (n *NetBus) sendToSocketByUniqId(uniqId string, data []byte) {
	if data == nil {
		return
	}
	if n.mainSocketManager != nil {
		mainSocket := n.mainSocketManager.GetSocket(UniqIdConvertToWorkerAddrAsHex(uniqId))
		if mainSocket != nil {
			mainSocket.Send(data)
			return
		}
	}
	taskSocket := n.taskSocketPoolManger.GetSocket(UniqIdConvertToWorkerAddrAsHex(uniqId))
	if taskSocket != nil {
		defer taskSocket.Release()
		taskSocket.Send(data)
		return
	}
}

func (n *NetBus) pack(cmd netsvrProtocol.Cmd, req proto.Message) []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data[0:4], uint32(cmd))
	if req == nil {
		return data
	}
	var err error
	data, err = (proto.MarshalOptions{}).MarshalAppend(data, req)
	if err != nil {
		logger.Error(fmt.Sprintf("Proto marshal %T failed", req), "error", err)
		return nil
	}
	return data
}