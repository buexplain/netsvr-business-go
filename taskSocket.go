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

import "time"

type TaskSocket struct {
	Socket
	pool *TaskSocketPool
}

func NewTaskSocket(workerAddr string, sendReceiveTimeout time.Duration, connectTimeout time.Duration, pool *TaskSocketPool) *TaskSocket {
	return &TaskSocket{
		Socket: Socket{
			workerAddr:         workerAddr,
			sendReceiveTimeout: sendReceiveTimeout,
			connectTimeout:     connectTimeout,
			connected:          new(int32),
		},
		pool: pool,
	}
}

func (t *TaskSocket) release() {
	if t.IsConnected() {
		t.pool.release(t)
	}
}
