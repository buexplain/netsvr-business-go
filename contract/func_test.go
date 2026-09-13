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

package contract

import "testing"

func TestFunc_AddrConvertToHex(t *testing.T) {
	if AddrConvertToHex("127.0.0.1:6061") != "7f00000117ad" {
		t.Error("task服务器监听的ip地址转为16进制字符串失败")
	}
}

func TestFunc_UniqIdConvertToAddrAsHex(t *testing.T) {
	// uniqId 的前 12 个十六进制字符就是网关的 task 服务地址
	uniqId := "7f00000117b8" + "68c7b2a1" + "00000001"
	if UniqIdConvertToAddrAsHex(uniqId) != "7f00000117b8" {
		t.Errorf("从 uniqId 解析网关地址失败：%s", UniqIdConvertToAddrAsHex(uniqId))
	}
	// 长度不是 28 的 uniqId 是不合法的
	if UniqIdConvertToAddrAsHex("7f00000117b8") != "" {
		t.Error("非法的 uniqId 应该返回空字符串")
	}
}
