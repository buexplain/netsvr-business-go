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

package ret

import (
	"slices"
	"testing"

	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
)

// assertSet 断言两个字符串列表的元素一致，忽略顺序（跨网关合并时 map 遍历顺序不确定）
func assertSet(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Fatalf("%s = %v, want 包含 %q", name, got, w)
		}
	}
}

func TestDedupAppend(t *testing.T) {
	seen := make(map[string]struct{})
	var got []string
	got = dedupAppend(got, seen, []string{"a", "b"})
	got = dedupAppend(got, seen, []string{"b", "c"})
	got = dedupAppend(got, seen, nil)
	if !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Fatalf("dedupAppend=%v want [a b c]，应去重并保持首次出现的顺序", got)
	}
}

func TestCheckOnlineRet(t *testing.T) {
	r := &CheckOnlineRet{Data: map[string]*netsvrProtocol.CheckOnlineResp{
		"gw1": {UniqIds: []string{"u1", "u2"}},
		"gw2": {UniqIds: []string{"u3"}},
	}}
	assertSet(t, "UniqIds", r.UniqIds(), []string{"u1", "u2", "u3"})
	if r.Len() != 3 {
		t.Fatalf("Len=%d want 3", r.Len())
	}
	if !r.Has("u3") {
		t.Fatal("Has(u3) 应返回 true")
	}
	if r.Has("u9") {
		t.Fatal("Has(u9) 应返回 false")
	}
}

func TestUniqIdListRet(t *testing.T) {
	r := &UniqIdListRet{Data: map[string]*netsvrProtocol.UniqIdListResp{
		"gw1": {UniqIds: []string{"u1", "u2"}},
		"gw2": {UniqIds: []string{"u3"}},
	}}
	assertSet(t, "UniqIds", r.UniqIds(), []string{"u1", "u2", "u3"})
	if r.Len() != 3 {
		t.Fatalf("Len=%d want 3", r.Len())
	}
	if !r.Has("u3") {
		t.Fatal("Has(u3) 应返回 true")
	}
	if r.Has("u9") {
		t.Fatal("Has(u9) 应返回 false")
	}
}

func TestCustomerIdListRet(t *testing.T) {
	r := &CustomerIdListRet{Data: map[string]*netsvrProtocol.CustomerIdListResp{
		"gw1": {CustomerIds: []string{"c1", "c2"}},
		"gw2": {CustomerIds: []string{"c2", "c3"}},
	}}
	// 同一个客户可能连接到多个网关，跨网关合并后需要去重
	assertSet(t, "CustomerIds", r.CustomerIds(), []string{"c1", "c2", "c3"})
	if r.Len() != 3 {
		t.Fatalf("Len=%d want 3", r.Len())
	}
	if !r.Has("c3") {
		t.Fatal("Has(c3) 应返回 true")
	}
	if r.Has("c9") {
		t.Fatal("Has(c9) 应返回 false")
	}
}

func TestTopicListRet(t *testing.T) {
	r := &TopicListRet{Data: map[string]*netsvrProtocol.TopicListResp{
		"gw1": {Topics: []string{"t1", "t2"}},
		"gw2": {Topics: []string{"t2", "t3"}},
	}}
	// 同名主题可能分布在多个网关，跨网关合并后需要去重
	assertSet(t, "Topics", r.Topics(), []string{"t1", "t2", "t3"})
	if !r.Has("t3") {
		t.Fatal("Has(t3) 应返回 true")
	}
	if r.Has("t9") {
		t.Fatal("Has(t9) 应返回 false")
	}
}

func TestTopicUniqIdListRet(t *testing.T) {
	r := &TopicUniqIdListRet{Data: map[string]*netsvrProtocol.TopicUniqIdListResp{
		"gw1": {Items: map[string]*netsvrProtocol.TopicUniqIdListRespItem{
			"t1": {UniqIds: []string{"u1"}},
		}},
		"gw2": {Items: map[string]*netsvrProtocol.TopicUniqIdListRespItem{
			"t1": {UniqIds: []string{"u2"}},
			"t2": {UniqIds: []string{"u3"}},
		}},
	}}
	// t1 的连接分布在两个网关，一个连接只属于一个网关，直接合并即可
	assertSet(t, "TopicUniqIds(t1)", r.TopicUniqIds("t1"), []string{"u1", "u2"})
	assertSet(t, "TopicUniqIds(t2)", r.TopicUniqIds("t2"), []string{"u3"})
	if r.TopicUniqIds("t404") != nil {
		t.Fatal("不存在的主题应返回 nil")
	}
	all := r.UniqIds()
	if len(all) != 2 {
		t.Fatalf("UniqIds()=%v want 2 个主题", all)
	}
	assertSet(t, "UniqIds()[t1]", all["t1"], []string{"u1", "u2"})
	assertSet(t, "UniqIds()[t2]", all["t2"], []string{"u3"})
	if _, ok := all["t404"]; ok {
		t.Fatal("不存在的主题不应出现在 UniqIds() 中")
	}
}

func TestTopicCustomerIdListRet(t *testing.T) {
	r := &TopicCustomerIdListRet{Data: map[string]*netsvrProtocol.TopicCustomerIdListResp{
		"gw1": {Items: map[string]*netsvrProtocol.TopicCustomerIdListRespItem{
			"t1": {CustomerIds: []string{"c1", "c2"}},
		}},
		"gw2": {Items: map[string]*netsvrProtocol.TopicCustomerIdListRespItem{
			"t1": {CustomerIds: []string{"c2", "c3"}},
			"t2": {CustomerIds: []string{"c4"}},
		}},
	}}
	// 同一个客户可能连接到多个网关，跨网关合并后需要去重
	assertSet(t, "TopicCustomerIds(t1)", r.TopicCustomerIds("t1"), []string{"c1", "c2", "c3"})
	assertSet(t, "TopicCustomerIds(t2)", r.TopicCustomerIds("t2"), []string{"c4"})
	if r.TopicCustomerIds("t404") != nil {
		t.Fatal("不存在的主题应返回 nil")
	}
	all := r.CustomerIds()
	if len(all) != 2 {
		t.Fatalf("CustomerIds()=%v want 2 个主题", all)
	}
	assertSet(t, "CustomerIds()[t1]", all["t1"], []string{"c1", "c2", "c3"})
	assertSet(t, "CustomerIds()[t2]", all["t2"], []string{"c4"})
	if _, ok := all["t404"]; ok {
		t.Fatal("不存在的主题不应出现在 CustomerIds() 中")
	}
}

func TestTopicCustomerIdToUniqIdsListRet(t *testing.T) {
	r := &TopicCustomerIdToUniqIdsListRet{Data: map[string]*netsvrProtocol.TopicCustomerIdToUniqIdsListResp{
		"gw1": {Items: map[string]*netsvrProtocol.TopicCustomerIdToUniqIdsListRespItem{
			"t1": {Items: map[string]*netsvrProtocol.CustomerIdToUniqIdsRespItem{
				"c1": {UniqIds: []string{"u1"}},
			}},
		}},
		"gw2": {Items: map[string]*netsvrProtocol.TopicCustomerIdToUniqIdsListRespItem{
			"t1": {Items: map[string]*netsvrProtocol.CustomerIdToUniqIdsRespItem{
				"c1": {UniqIds: []string{"u2"}},
				"c2": {UniqIds: []string{"u3"}},
			}},
		}},
	}}
	// 同一个客户可能连接到多个网关，跨网关合并后需要去重
	assertSet(t, "TopicCustomerIds(t1)", r.TopicCustomerIds("t1"), []string{"c1", "c2"})
	if r.TopicCustomerIds("t404") != nil {
		t.Fatal("不存在的主题应返回 nil")
	}
	// 该客户的连接分布在两个网关，一个连接只属于一个网关，直接合并即可
	assertSet(t, "CustomerUniqIds(t1,c1)", r.CustomerUniqIds("t1", "c1"), []string{"u1", "u2"})
	assertSet(t, "CustomerUniqIds(t1,c2)", r.CustomerUniqIds("t1", "c2"), []string{"u3"})
	if r.CustomerUniqIds("t1", "c9") != nil {
		t.Fatal("不存在的客户应返回 nil")
	}
	if r.CustomerUniqIds("t404", "c1") != nil {
		t.Fatal("不存在的主题应返回 nil")
	}
}

func TestConnInfoRet(t *testing.T) {
	r := &ConnInfoRet{Data: map[string]*netsvrProtocol.ConnInfoResp{
		"gw1": {Items: map[string]*netsvrProtocol.ConnInfoRespItem{
			"u1": {Session: "s1"},
		}},
		"gw2": {Items: map[string]*netsvrProtocol.ConnInfoRespItem{
			"u2": {Session: "s2"},
		}},
	}}
	// 一个连接只属于一个网关，命中即返回
	item, ok := r.Get("u2")
	if !ok || item.GetSession() != "s2" {
		t.Fatalf("Get(u2)=%v,%v want s2,true", item, ok)
	}
	if _, ok := r.Get("u9"); ok {
		t.Fatal("Get(u9) 不应命中")
	}
	all := r.ToMap()
	if len(all) != 2 || all["u1"].GetSession() != "s1" || all["u2"].GetSession() != "s2" {
		t.Fatalf("ToMap()=%v", all)
	}
}

func TestConnInfoByCustomerIdRet(t *testing.T) {
	r := &ConnInfoByCustomerIdRet{Data: map[string]*netsvrProtocol.ConnInfoByCustomerIdResp{
		"gw1": {Items: map[string]*netsvrProtocol.ConnInfoByCustomerIdRespItems{
			"c1": {Items: []*netsvrProtocol.ConnInfoByCustomerIdRespItem{{UniqId: "u1"}}},
		}},
		"gw2": {Items: map[string]*netsvrProtocol.ConnInfoByCustomerIdRespItems{
			"c1": {Items: []*netsvrProtocol.ConnInfoByCustomerIdRespItem{{UniqId: "u2"}}},
			"c2": {Items: []*netsvrProtocol.ConnInfoByCustomerIdRespItem{{UniqId: "u3"}}},
		}},
	}}
	// 同一个客户可能连接到多个网关，各网关的连接会合并
	got := r.Get("c1")
	if len(got) != 2 {
		t.Fatalf("Get(c1)=%v want 2 条", got)
	}
	uniqIds := make([]string, 0, len(got))
	for _, item := range got {
		uniqIds = append(uniqIds, item.GetUniqId())
	}
	assertSet(t, "Get(c1)", uniqIds, []string{"u1", "u2"})
	if r.Get("c9") != nil {
		t.Fatal("不存在的客户应返回 nil")
	}
	all := r.ToMap()
	if len(all) != 2 {
		t.Fatalf("ToMap()=%v want 2 个客户", all)
	}
	if len(all["c1"]) != 2 || len(all["c2"]) != 1 {
		t.Fatalf("ToMap()=%v", all)
	}
	if _, ok := all["c9"]; ok {
		t.Fatal("不存在的客户不应出现在 ToMap() 中")
	}
}

func TestUniqIdCountRet(t *testing.T) {
	r := &UniqIdCountRet{Data: map[string]*netsvrProtocol.UniqIdCountResp{
		"gw1": {Count: 2},
		"gw2": {Count: 3},
	}}
	// 一个连接只属于一个网关，各网关的计数不会重复，直接相加即可
	if r.Count() != 5 {
		t.Fatalf("Count=%d want 5", r.Count())
	}
	if (&UniqIdCountRet{}).Count() != 0 {
		t.Fatal("空结果 Count 应为 0")
	}
}

func TestTopicCountRet(t *testing.T) {
	r := &TopicCountRet{Data: map[string]*netsvrProtocol.TopicCountResp{
		"gw1": {Count: 2},
		"gw2": {Count: 3},
	}}
	// 多网关部署时不同网关之间的同名主题会被重复统计，因此直接相加
	if r.Count() != 5 {
		t.Fatalf("Count=%d want 5", r.Count())
	}
	if (&TopicCountRet{}).Count() != 0 {
		t.Fatal("空结果 Count 应为 0")
	}
}

// TestEmptyData 各方法在没有任何网关返回结果时，应返回零值且不 panic
func TestEmptyData(t *testing.T) {
	if r := (&CheckOnlineRet{}); r.Len() != 0 || len(r.UniqIds()) != 0 || r.Has("u1") {
		t.Fatal("CheckOnlineRet 空结果应返回零值")
	}
	if r := (&UniqIdListRet{}); r.Len() != 0 || len(r.UniqIds()) != 0 || r.Has("u1") {
		t.Fatal("UniqIdListRet 空结果应返回零值")
	}
	if r := (&CustomerIdListRet{}); r.Len() != 0 || len(r.CustomerIds()) != 0 || r.Has("c1") {
		t.Fatal("CustomerIdListRet 空结果应返回零值")
	}
	if r := (&TopicListRet{}); len(r.Topics()) != 0 || r.Has("t1") {
		t.Fatal("TopicListRet 空结果应返回零值")
	}
	if r := (&TopicUniqIdListRet{}); r.TopicUniqIds("t1") != nil || len(r.UniqIds()) != 0 {
		t.Fatal("TopicUniqIdListRet 空结果应返回零值")
	}
	if r := (&TopicCustomerIdListRet{}); r.TopicCustomerIds("t1") != nil || len(r.CustomerIds()) != 0 {
		t.Fatal("TopicCustomerIdListRet 空结果应返回零值")
	}
	if r := (&TopicCustomerIdToUniqIdsListRet{}); r.TopicCustomerIds("t1") != nil || r.CustomerUniqIds("t1", "c1") != nil {
		t.Fatal("TopicCustomerIdToUniqIdsListRet 空结果应返回零值")
	}
	if r := (&ConnInfoRet{}); len(r.ToMap()) != 0 {
		t.Fatal("ConnInfoRet 空结果应返回零值")
	} else if _, ok := r.Get("u1"); ok {
		t.Fatal("ConnInfoRet 空结果 Get 不应命中")
	}
	if r := (&ConnInfoByCustomerIdRet{}); len(r.ToMap()) != 0 || r.Get("c1") != nil {
		t.Fatal("ConnInfoByCustomerIdRet 空结果应返回零值")
	}
}
