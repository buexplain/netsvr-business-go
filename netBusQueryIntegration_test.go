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

	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
)

// setUniqueCustomerId 让每个连接使用自己的 uniqId 作为 customerId，便于断言
func (e *env) setUniqueCustomerId(uniqIds []string) {
	e.t.Helper()
	e.updateConnInfo(uniqIds, func(uniqId string, up *netsvrProtocol.ConnInfoUpdate) {
		up.NewCustomerId = uniqId
	})
}

// shareCustomerId 让每个网关各挑一个连接共享同一个 customerId，返回共享的 customerId 与这些连接
func (e *env) shareCustomerId(uniqIds []string) (string, []string) {
	e.t.Helper()
	sharedCustomerId := uniqueData("sharedCustomerId")
	sharedUniqIds := make([]string, 0, len(e.gateways))
	for _, gw := range e.gateways {
		gwUniqIds := e.uniqIdsByTaskAddr(gw.taskAddr)
		sharedUniqIds = append(sharedUniqIds, gwUniqIds[0])
		e.updateConnInfo(gwUniqIds[:1], func(uniqId string, up *netsvrProtocol.ConnInfoUpdate) {
			up.NewCustomerId = sharedCustomerId
		})
	}
	return sharedCustomerId, sharedUniqIds
}

// subscribeAll 让全部连接订阅给定的主题
func (e *env) subscribeAll(uniqIds []string, topics []string) {
	e.t.Helper()
	for _, uniqId := range uniqIds {
		e.bus.TopicSubscribe(uniqId, topics, nil)
	}
}

// TestCheckOnline 检查连接是否在线
func TestCheckOnline(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		ret := e.bus.CheckOnline(uniqIds)
		// 并行跑其它包的测试时，网关里可能还有别的连接，这里只断言本次用例的连接都在线
		for _, uniqId := range uniqIds {
			if !ret.Has(uniqId) {
				t.Fatalf("CheckOnline 未命中在线的连接 %s", uniqId)
			}
		}
		if ret.Len() < len(uniqIds) {
			t.Fatalf("CheckOnline 的在线数量：至少 %d，实际 %d", len(uniqIds), ret.Len())
		}
		if ret.Has("notExistUniqId") {
			t.Fatalf("CheckOnline 不应命中不存在的连接")
		}
	})
}

// TestUniqIdList 获取网关中的全部连接
func TestUniqIdList(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		ret := e.bus.UniqIdList()
		// 并行跑其它包的测试时，网关里可能还有别的连接，这里只断言本次用例的连接都在列表里
		for _, uniqId := range uniqIds {
			if !ret.Has(uniqId) {
				t.Fatalf("UniqIdList 未命中在线的连接 %s", uniqId)
			}
		}
		if ret.Len() < len(uniqIds) {
			t.Fatalf("UniqIdList 的连接数量：至少 %d，实际 %d", len(uniqIds), ret.Len())
		}
		if ret.Has("notExistUniqId") {
			t.Fatalf("UniqIdList 不应命中不存在的连接")
		}
	})
}

// TestUniqIdCount 统计在线连接数：一个连接只属于一个网关，多网关直接相加
func TestUniqIdCount(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		// 并行跑其它包的测试时，网关里可能还有别的连接，这里断言的是下界
		if got := e.bus.UniqIdCount().Count(); got < int32(len(uniqIds)) {
			t.Fatalf("UniqIdCount：至少 %d，实际 %d", len(uniqIds), got)
		}
	})
}

// TestCustomerIdList 获取在线客户：同一个客户可能连接到多个网关，跨网关去重
func TestCustomerIdList(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		e.setUniqueCustomerId(uniqIds)
		sharedCustomerId, sharedUniqIds := e.shareCustomerId(uniqIds)
		// 期望的客户列表：共享的客户只算一个，其余连接各自一个
		shared := make(map[string]struct{}, len(sharedUniqIds))
		for _, uniqId := range sharedUniqIds {
			shared[uniqId] = struct{}{}
		}
		expectCustomerIds := make([]string, 0, len(uniqIds))
		expectCustomerIds = append(expectCustomerIds, sharedCustomerId)
		for _, uniqId := range uniqIds {
			if _, ok := shared[uniqId]; ok {
				continue
			}
			expectCustomerIds = append(expectCustomerIds, uniqId)
		}
		ret := e.bus.CustomerIdList()
		assertSameSet(t, "CustomerIdList 的 customerId", expectCustomerIds, ret.CustomerIds())
		if ret.Len() != len(expectCustomerIds) {
			t.Fatalf("CustomerIdList 的去重后客户数：期望 %d，实际 %d", len(expectCustomerIds), ret.Len())
		}
		if !ret.Has(sharedCustomerId) {
			t.Fatalf("CustomerIdList 未命中共享的客户 %s", sharedCustomerId)
		}
		if ret.Has("notExistCustomerId") {
			t.Fatalf("CustomerIdList 不应命中不存在的客户")
		}
	})
}

// TestCustomerIdCount 统计在线客户数：每个网关各自统计自己的连接
func TestCustomerIdCount(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		e.setUniqueCustomerId(uniqIds)
		ret := e.bus.CustomerIdCount()
		if len(ret.Data) != len(gateways) {
			t.Fatalf("CustomerIdCount 返回的网关数量：期望 %d，实际 %d", len(gateways), len(ret.Data))
		}
		var total int32
		for addr, resp := range ret.Data {
			if resp.GetCount() != onlineNumPerGateway {
				t.Fatalf("网关 %s 的客户数：期望 %d，实际 %d", addr, onlineNumPerGateway, resp.GetCount())
			}
			total += resp.GetCount()
		}
		if total != int32(len(uniqIds)) {
			t.Fatalf("CustomerIdCount 汇总：期望 %d，实际 %d", len(uniqIds), total)
		}
	})
}

// TestTopicList 获取网关中的主题
func TestTopicList(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		topics := []string{uniqueData("topic"), uniqueData("topic")}
		e.subscribeAll(uniqIds, topics)
		ret := e.bus.TopicList()
		assertSameSet(t, "TopicList 的 topic", topics, ret.Topics())
		if !ret.Has(topics[0]) {
			t.Fatalf("TopicList 未命中主题 %s", topics[0])
		}
		if ret.Has("notExistTopic") {
			t.Fatalf("TopicList 不应命中不存在的主题")
		}
	})
}

// TestTopicCount 统计主题数：多网关部署时不同网关之间的同名主题会被重复统计
func TestTopicCount(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		topics := []string{uniqueData("topic"), uniqueData("topic")}
		e.subscribeAll(uniqIds, topics)
		expect := int32(len(topics) * len(gateways))
		if got := e.bus.TopicCount().Count(); got != expect {
			t.Fatalf("TopicCount：期望 %d（主题数 %d × 网关数 %d），实际 %d", expect, len(topics), len(gateways), got)
		}
	})
}

// TestTopicUniqIdList 获取主题包含的连接
func TestTopicUniqIdList(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		topics := []string{uniqueData("topic"), uniqueData("topic")}
		e.subscribeAll(uniqIds, topics)
		ret := e.bus.TopicUniqIdList(topics)
		for _, topic := range topics {
			assertSameSet(t, "TopicUniqIdList 的 uniqId", uniqIds, ret.TopicUniqIds(topic))
		}
		if got := ret.TopicUniqIds("notExistTopic"); len(got) != 0 {
			t.Fatalf("不存在的主题应返回空：%v", got)
		}
		all := ret.UniqIds()
		if len(all) != len(topics) {
			t.Fatalf("TopicUniqIdList 的主题数量：期望 %d，实际 %d", len(topics), len(all))
		}
		for _, topic := range topics {
			assertSameSet(t, "TopicUniqIdList 的 UniqIds", uniqIds, all[topic])
		}
	})
}

// TestTopicUniqIdCount 统计主题包含的连接数：多网关相加会重复统计同名主题
func TestTopicUniqIdCount(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		topics := []string{uniqueData("topic"), uniqueData("topic")}
		e.subscribeAll(uniqIds, topics)
		ret := e.bus.TopicUniqIdCount(topics)
		if len(ret.Data) != len(gateways) {
			t.Fatalf("TopicUniqIdCount 返回的网关数量：期望 %d，实际 %d", len(gateways), len(ret.Data))
		}
		for addr, resp := range ret.Data {
			for _, topic := range topics {
				if got := resp.GetItems()[topic]; got != onlineNumPerGateway {
					t.Fatalf("网关 %s 主题 %s 的连接数：期望 %d，实际 %d", addr, topic, onlineNumPerGateway, got)
				}
			}
		}
	})
}

// TestTopicCustomerIdList 获取主题包含的客户
func TestTopicCustomerIdList(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		e.setUniqueCustomerId(uniqIds)
		topics := []string{uniqueData("topic"), uniqueData("topic")}
		e.subscribeAll(uniqIds, topics)
		ret := e.bus.TopicCustomerIdList(topics)
		for _, topic := range topics {
			assertSameSet(t, "TopicCustomerIdList 的 customerId", uniqIds, ret.TopicCustomerIds(topic))
		}
		if got := ret.TopicCustomerIds("notExistTopic"); len(got) != 0 {
			t.Fatalf("不存在的主题应返回空：%v", got)
		}
		all := ret.CustomerIds()
		if len(all) != len(topics) {
			t.Fatalf("TopicCustomerIdList 的主题数量：期望 %d，实际 %d", len(topics), len(all))
		}
		for _, topic := range topics {
			assertSameSet(t, "TopicCustomerIdList 的 CustomerIds", uniqIds, all[topic])
		}
	})
}

// TestTopicCustomerIdCount 统计主题包含的客户数
func TestTopicCustomerIdCount(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		e.setUniqueCustomerId(uniqIds)
		topics := []string{uniqueData("topic"), uniqueData("topic")}
		e.subscribeAll(uniqIds, topics)
		ret := e.bus.TopicCustomerIdCount(topics)
		if len(ret.Data) != len(gateways) {
			t.Fatalf("TopicCustomerIdCount 返回的网关数量：期望 %d，实际 %d", len(gateways), len(ret.Data))
		}
		for addr, resp := range ret.Data {
			for _, topic := range topics {
				if got := resp.GetItems()[topic]; got != onlineNumPerGateway {
					t.Fatalf("网关 %s 主题 %s 的客户数：期望 %d，实际 %d", addr, topic, onlineNumPerGateway, got)
				}
			}
		}
	})
}

// TestTopicCustomerIdToUniqIdsList 获取主题下的客户及其全部连接
func TestTopicCustomerIdToUniqIdsList(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		e.setUniqueCustomerId(uniqIds)
		topics := []string{uniqueData("topic"), uniqueData("topic")}
		e.subscribeAll(uniqIds, topics)
		ret := e.bus.TopicCustomerIdToUniqIdsList(topics)
		for _, topic := range topics {
			assertSameSet(t, "TopicCustomerIdToUniqIdsList 的 customerId", uniqIds, ret.TopicCustomerIds(topic))
			// 每个客户只有一个连接
			for _, uniqId := range uniqIds {
				assertSameSet(t, "TopicCustomerIdToUniqIdsList 的 uniqId", []string{uniqId}, ret.CustomerUniqIds(topic, uniqId))
			}
		}
		if got := ret.TopicCustomerIds("notExistTopic"); len(got) != 0 {
			t.Fatalf("不存在的主题应返回空：%v", got)
		}
		if got := ret.CustomerUniqIds("notExistTopic", "notExistCustomerId"); len(got) != 0 {
			t.Fatalf("不存在的目标应返回空：%v", got)
		}
	})
}

// TestConnInfoGet 获取连接存储的信息，并覆盖各个字段的按需获取
func TestConnInfoGet(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		// 还没有存储任何信息
		ret := e.bus.ConnInfo(uniqIds, true, true, true)
		for _, uniqId := range uniqIds {
			item, ok := ret.Get(uniqId)
			if !ok {
				t.Fatalf("连接 %s 的信息不存在", uniqId)
			}
			if item.GetSession() != "" || item.GetCustomerId() != "" || len(item.GetTopics()) != 0 {
				t.Fatalf("连接 %s 初始不应该有任何信息：session=%q customerId=%q topics=%v", uniqId, item.GetSession(), item.GetCustomerId(), item.GetTopics())
			}
		}
		if _, ok := ret.Get("notExistUniqId"); ok {
			t.Fatalf("不存在的 uniqId 不应命中")
		}
		// 设置信息
		topics := []string{uniqueData("topic")}
		e.updateConnInfo(uniqIds, func(uniqId string, up *netsvrProtocol.ConnInfoUpdate) {
			up.NewSession = uniqId + "Session"
			up.NewCustomerId = uniqId + "CustomerId"
			up.NewTopics = topics
		})
		// 只获取 customerId
		onlyCustomerId := e.bus.ConnInfo(uniqIds, false, true, false)
		for _, uniqId := range uniqIds {
			item, ok := onlyCustomerId.Get(uniqId)
			if !ok {
				t.Fatalf("连接 %s 的信息不存在", uniqId)
			}
			if item.GetCustomerId() != uniqId+"CustomerId" {
				t.Fatalf("连接 %s 的 customerId 不符合预期：%q", uniqId, item.GetCustomerId())
			}
			if item.GetSession() != "" || len(item.GetTopics()) != 0 {
				t.Fatalf("连接 %s 不应该返回 session 与 topic：session=%q topics=%v", uniqId, item.GetSession(), item.GetTopics())
			}
		}
		// 只获取 session
		onlySession := e.bus.ConnInfo(uniqIds, true, false, false)
		for _, uniqId := range uniqIds {
			item, ok := onlySession.Get(uniqId)
			if !ok {
				t.Fatalf("连接 %s 的信息不存在", uniqId)
			}
			if item.GetSession() != uniqId+"Session" {
				t.Fatalf("连接 %s 的 session 不符合预期：%q", uniqId, item.GetSession())
			}
			if item.GetCustomerId() != "" || len(item.GetTopics()) != 0 {
				t.Fatalf("连接 %s 不应该返回 customerId 与 topic：customerId=%q topics=%v", uniqId, item.GetCustomerId(), item.GetTopics())
			}
		}
		// 只获取 topic
		onlyTopic := e.bus.ConnInfo(uniqIds, false, false, true)
		for _, uniqId := range uniqIds {
			item, ok := onlyTopic.Get(uniqId)
			if !ok {
				t.Fatalf("连接 %s 的信息不存在", uniqId)
			}
			assertSameSet(t, "连接的主题", topics, item.GetTopics())
			if item.GetSession() != "" || item.GetCustomerId() != "" {
				t.Fatalf("连接 %s 不应该返回 session 与 customerId：session=%q customerId=%q", uniqId, item.GetSession(), item.GetCustomerId())
			}
		}
	})
}

// TestConnInfoByCustomerId 获取客户存储的信息：同一个客户可能连接到多个网关，各网关的连接会合并
func TestConnInfoByCustomerId(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		e.setUniqueCustomerId(uniqIds)
		sharedCustomerId, sharedUniqIds := e.shareCustomerId(uniqIds)
		ret := e.bus.ConnInfoByCustomerId([]string{sharedCustomerId}, true, true, true)
		items := ret.Get(sharedCustomerId)
		gotUniqIds := make([]string, 0, len(items))
		for _, item := range items {
			gotUniqIds = append(gotUniqIds, item.GetUniqId())
		}
		assertSameSet(t, "ConnInfoByCustomerId 的 uniqId", sharedUniqIds, gotUniqIds)
		if got := ret.Get("notExistCustomerId"); len(got) != 0 {
			t.Fatalf("不存在的客户应返回空：%v", got)
		}
		if got := len(ret.ToMap()[sharedCustomerId]); got != len(sharedUniqIds) {
			t.Fatalf("ToMap 中共享客户的连接数：期望 %d，实际 %d", len(sharedUniqIds), got)
		}
	})
}

// TestMetrics 获取网关的统计信息
func TestMetrics(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		ret := e.bus.Metrics()
		if len(ret.Data) != len(gateways) {
			t.Fatalf("Metrics 返回的网关数量：期望 %d，实际 %d", len(gateways), len(ret.Data))
		}
		for addr, resp := range ret.Data {
			if len(resp.GetItems()) == 0 {
				t.Fatalf("网关 %s 的统计项为空", addr)
			}
		}
	})
}

// TestLimit 读取与设置网关的限流配置
func TestLimit(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		// 读取全部网关的限流配置
		ret := e.bus.Limit(nil, "")
		if ret == nil {
			t.Fatalf("Limit 返回 nil")
		}
		if len(ret.Data) != len(gateways) {
			t.Fatalf("Limit 返回的网关数量：期望 %d，实际 %d", len(gateways), len(ret.Data))
		}
		for addr, resp := range ret.Data {
			if resp.GetOnOpen() == 0 || resp.GetOnMessage() == 0 {
				t.Fatalf("网关 %s 的限流配置不符合预期：onOpen=%d onMessage=%d", addr, resp.GetOnOpen(), resp.GetOnMessage())
			}
		}
		// 只读取指定网关的限流配置
		single := e.bus.Limit(nil, gateways[0].taskAddr)
		if single == nil {
			t.Fatalf("Limit 指定网关时返回 nil")
		}
		if len(single.Data) != 1 {
			t.Fatalf("Limit 指定网关时返回的网关数量：期望 1，实际 %d", len(single.Data))
		}
		// 不存在的网关返回 nil
		if got := e.bus.Limit(nil, "127.0.0.1:1"); got != nil {
			t.Fatalf("Limit 指定不存在的网关应返回 nil，实际 %v", got)
		}
	})
}
