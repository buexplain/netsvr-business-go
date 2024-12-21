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

import "testing"

func TestFunc_WorkerAddrConvertToHex(t *testing.T) {
	if WorkerAddrConvertToHex("127.0.0.1:6061") != "7f00000117ad" {
		t.Error("网关的worker服务器监听的地址转为16进制字符串失败")
	}
}
