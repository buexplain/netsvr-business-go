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
	"strings"
	"testing"
	"time"

	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
)

// TestBroadcast 广播：单条广播与批量广播
func TestBroadcast(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		// 单条广播
		message := uniqueData("broadcast")
		e.bus.Broadcast([]byte(message))
		for _, client := range e.allClients() {
			if got := client.receive(t); got != message {
				t.Fatalf("连接 %s 收到的广播数据不符合预期：期望 %q，实际 %q", client.uniqId, message, got)
			}
		}
		// 批量广播：每个连接按顺序收到全部数据
		dataList := []string{uniqueData("broadcastBulk"), uniqueData("broadcastBulk")}
		e.bus.BroadcastBulk([][]byte{[]byte(dataList[0]), []byte(dataList[1])})
		for _, client := range e.allClients() {
			for _, want := range dataList {
				if got := client.receive(t); got != want {
					t.Fatalf("连接 %s 收到的批量广播数据不符合预期：期望 %q，实际 %q", client.uniqId, want, got)
				}
			}
		}
	})
}

// TestSendToUniqId 按 uniqId 单播，包含大包场景
func TestSendToUniqId(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		// 每个连接各收到一条不同的数据
		messageOf := make(map[string]string, onlineNumPerGateway*len(gateways))
		for _, client := range e.allClients() {
			messageOf[client.uniqId] = uniqueData("sendToUniqId")
		}
		for uniqId, message := range messageOf {
			e.bus.SendToUniqId(uniqId, []byte(message))
		}
		for _, client := range e.allClients() {
			if got := client.receive(t); got != messageOf[client.uniqId] {
				t.Fatalf("连接 %s 收到的单播数据不符合预期：期望 %q，实际 %q", client.uniqId, messageOf[client.uniqId], got)
			}
		}
		// 大包：约 229KB，覆盖底层大帧的读取
		client := e.allClients()[0]
		largeData := strings.Repeat("a", 65536*3+32768)
		e.bus.SendToUniqId(client.uniqId, []byte(largeData))
		if got := client.receive(t); got != largeData {
			t.Fatalf("连接 %s 收到的大包数据不符合预期：长度期望 %d，实际 %d", client.uniqId, len(largeData), len(got))
		}
	})
}

// TestSendToUniqIds 按 uniqId 组播：一组连接收到同一条数据
func TestSendToUniqIds(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		message := uniqueData("sendToUniqIds")
		e.bus.SendToUniqIds(e.uniqIds(), []byte(message))
		for _, client := range e.allClients() {
			if got := client.receive(t); got != message {
				t.Fatalf("连接 %s 收到的组播数据不符合预期：期望 %q，实际 %q", client.uniqId, message, got)
			}
		}
	})
}

// TestSingleCastBulk 按 uniqId 批量单播：覆盖三种目标与数据的组合
func TestSingleCastBulk(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		// 每一项一个目标、一条数据
		messageOf := make(map[string]string, len(uniqIds))
		items := make([]*netsvrProtocol.SingleCastBulkItem, 0, len(uniqIds))
		for _, uniqId := range uniqIds {
			message := uniqueData("singleCastBulk")
			messageOf[uniqId] = message
			item := &netsvrProtocol.SingleCastBulkItem{UniqIds: []string{uniqId}, Data: [][]byte{[]byte(message)}}
			items = append(items, item)
		}
		e.bus.SingleCastBulk(items)
		for _, client := range e.allClients() {
			if got := client.receive(t); got != messageOf[client.uniqId] {
				t.Fatalf("连接 %s 收到的批量单播数据不符合预期：期望 %q，实际 %q", client.uniqId, messageOf[client.uniqId], got)
			}
		}
		// 一项内多个目标共享同一条数据
		message := uniqueData("singleCastBulk")
		item := &netsvrProtocol.SingleCastBulkItem{UniqIds: uniqIds, Data: [][]byte{[]byte(message)}}
		e.bus.SingleCastBulk([]*netsvrProtocol.SingleCastBulkItem{item})
		for _, client := range e.allClients() {
			if got := client.receive(t); got != message {
				t.Fatalf("连接 %s 收到的批量单播数据不符合预期：期望 %q，实际 %q", client.uniqId, message, got)
			}
		}
		// 一项内一个目标接收多条数据
		targetUniqId := uniqIds[0]
		dataList := []string{uniqueData("singleCastBulk"), uniqueData("singleCastBulk")}
		item = &netsvrProtocol.SingleCastBulkItem{UniqIds: []string{targetUniqId}, Data: [][]byte{[]byte(dataList[0]), []byte(dataList[1])}}
		e.bus.SingleCastBulk([]*netsvrProtocol.SingleCastBulkItem{item})
		for _, client := range e.allClients() {
			if client.uniqId != targetUniqId {
				continue
			}
			for _, want := range dataList {
				if got := client.receive(t); got != want {
					t.Fatalf("连接 %s 收到的批量单播数据不符合预期：期望 %q，实际 %q", client.uniqId, want, got)
				}
			}
		}
		// 一项内多个目标 × 多条数据：每个目标都会按顺序收到全部数据
		dataList = []string{uniqueData("singleCastBulk"), uniqueData("singleCastBulk")}
		item = &netsvrProtocol.SingleCastBulkItem{UniqIds: uniqIds, Data: [][]byte{[]byte(dataList[0]), []byte(dataList[1])}}
		e.bus.SingleCastBulk([]*netsvrProtocol.SingleCastBulkItem{item})
		for _, client := range e.allClients() {
			for _, want := range dataList {
				if got := client.receive(t); got != want {
					t.Fatalf("连接 %s 收到的批量单播数据不符合预期：期望 %q，实际 %q", client.uniqId, want, got)
				}
			}
		}
	})
}

// TestSendToCustomerId 按 customerId 单播
func TestSendToCustomerId(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		// 每个连接使用独立的 customerId
		messageOf := make(map[string]string, len(uniqIds))
		e.updateConnInfo(uniqIds, func(uniqId string, up *netsvrProtocol.ConnInfoUpdate) {
			message := uniqueData("sendToCustomerId")
			messageOf[uniqId] = message
			up.NewCustomerId = uniqId
		})
		for _, uniqId := range uniqIds {
			e.bus.SendToCustomerId(uniqId, []byte(messageOf[uniqId]))
		}
		for _, client := range e.allClients() {
			if got := client.receive(t); got != messageOf[client.uniqId] {
				t.Fatalf("连接 %s 收到的单播数据不符合预期：期望 %q，实际 %q", client.uniqId, messageOf[client.uniqId], got)
			}
		}
	})
}

// TestSendToCustomerIds 按 customerId 组播：一组客户的所有连接收到同一条数据
func TestSendToCustomerIds(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		e.updateConnInfo(uniqIds, func(uniqId string, up *netsvrProtocol.ConnInfoUpdate) {
			up.NewCustomerId = uniqId
		})
		message := uniqueData("sendToCustomerIds")
		e.bus.SendToCustomerIds(uniqIds, []byte(message))
		for _, client := range e.allClients() {
			if got := client.receive(t); got != message {
				t.Fatalf("连接 %s 收到的组播数据不符合预期：期望 %q，实际 %q", client.uniqId, message, got)
			}
		}
	})
}

// TestSingleCastBulkByCustomerId 按 customerId 批量单播：覆盖三种目标与数据的组合
func TestSingleCastBulkByCustomerId(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		// 每个连接使用独立的 customerId
		e.updateConnInfo(uniqIds, func(uniqId string, up *netsvrProtocol.ConnInfoUpdate) {
			up.NewCustomerId = uniqId
		})
		// 每一项一个客户、一条数据
		messageOf := make(map[string]string, len(uniqIds))
		items := make([]*netsvrProtocol.SingleCastBulkByCustomerIdItem, 0, len(uniqIds))
		for _, uniqId := range uniqIds {
			message := uniqueData("singleCastBulkByCustomerId")
			messageOf[uniqId] = message
			item := &netsvrProtocol.SingleCastBulkByCustomerIdItem{CustomerIds: []string{uniqId}, Data: [][]byte{[]byte(message)}}
			items = append(items, item)
		}
		e.bus.SingleCastBulkByCustomerId(items)
		for _, client := range e.allClients() {
			if got := client.receive(t); got != messageOf[client.uniqId] {
				t.Fatalf("连接 %s 收到的批量单播数据不符合预期：期望 %q，实际 %q", client.uniqId, messageOf[client.uniqId], got)
			}
		}
		// 一项内多个客户共享同一条数据
		message := uniqueData("singleCastBulkByCustomerId")
		item := &netsvrProtocol.SingleCastBulkByCustomerIdItem{CustomerIds: uniqIds, Data: [][]byte{[]byte(message)}}
		e.bus.SingleCastBulkByCustomerId([]*netsvrProtocol.SingleCastBulkByCustomerIdItem{item})
		for _, client := range e.allClients() {
			if got := client.receive(t); got != message {
				t.Fatalf("连接 %s 收到的批量单播数据不符合预期：期望 %q，实际 %q", client.uniqId, message, got)
			}
		}
		// 一项内多个客户 × 多条数据：每个客户都会按顺序收到全部数据
		dataList := []string{uniqueData("singleCastBulkByCustomerId"), uniqueData("singleCastBulkByCustomerId")}
		item = &netsvrProtocol.SingleCastBulkByCustomerIdItem{CustomerIds: uniqIds, Data: [][]byte{[]byte(dataList[0]), []byte(dataList[1])}}
		e.bus.SingleCastBulkByCustomerId([]*netsvrProtocol.SingleCastBulkByCustomerIdItem{item})
		for _, client := range e.allClients() {
			for _, want := range dataList {
				if got := client.receive(t); got != want {
					t.Fatalf("连接 %s 收到的批量单播数据不符合预期：期望 %q，实际 %q", client.uniqId, want, got)
				}
			}
		}
	})
}

// TestTopicSubscribeUnsubscribe 订阅、查看订阅、取消订阅
func TestTopicSubscribeUnsubscribe(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		topics := []string{uniqueData("topic"), uniqueData("topic")}
		for _, uniqId := range uniqIds {
			e.bus.TopicSubscribe(uniqId, topics, nil)
		}
		ret := e.bus.ConnInfo(uniqIds, false, false, true)
		for _, uniqId := range uniqIds {
			item, ok := ret.Get(uniqId)
			if !ok {
				t.Fatalf("连接 %s 的订阅信息不存在", uniqId)
			}
			assertSameSet(t, "订阅后的主题", topics, item.GetTopics())
		}
		for _, uniqId := range uniqIds {
			e.bus.TopicUnsubscribe(uniqId, topics, nil)
		}
		ret = e.bus.ConnInfo(uniqIds, false, false, true)
		for _, uniqId := range uniqIds {
			item, ok := ret.Get(uniqId)
			if !ok {
				t.Fatalf("连接 %s 的信息不存在", uniqId)
			}
			if len(item.GetTopics()) != 0 {
				t.Fatalf("连接 %s 取消订阅后仍有主题：%v", uniqId, item.GetTopics())
			}
		}
	})
}

// TestTopicDelete 删除主题后，连接上存储的主题也会被清除
func TestTopicDelete(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		topics := []string{uniqueData("topic"), uniqueData("topic")}
		for _, uniqId := range uniqIds {
			e.bus.TopicSubscribe(uniqId, topics, nil)
		}
		e.bus.TopicDelete(topics, nil)
		ret := e.bus.ConnInfo(uniqIds, false, false, true)
		for _, uniqId := range uniqIds {
			item, ok := ret.Get(uniqId)
			if !ok {
				t.Fatalf("连接 %s 的信息不存在", uniqId)
			}
			if len(item.GetTopics()) != 0 {
				t.Fatalf("连接 %s 的主题被删除后仍有主题：%v", uniqId, item.GetTopics())
			}
		}
	})
}

// TestPublishToTopic 给单个主题发布一条数据
func TestPublishToTopic(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		topic := uniqueData("topic")
		for _, uniqId := range uniqIds {
			e.bus.TopicSubscribe(uniqId, []string{topic}, nil)
		}
		message := uniqueData("publishToTopic")
		e.bus.PublishToTopic(topic, []byte(message))
		for _, client := range e.allClients() {
			if got := client.receive(t); got != message {
				t.Fatalf("连接 %s 收到的发布数据不符合预期：期望 %q，实际 %q", client.uniqId, message, got)
			}
		}
	})
}

// TestPublishToTopics 给一组主题发布同一条数据，订阅了多个主题的连接会收到多条
func TestPublishToTopics(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		topics := []string{uniqueData("topic"), uniqueData("topic")}
		for _, uniqId := range uniqIds {
			e.bus.TopicSubscribe(uniqId, topics, nil)
		}
		message := uniqueData("publishToTopics")
		e.bus.PublishToTopics(topics, []byte(message))
		for _, client := range e.allClients() {
			for range topics {
				if got := client.receive(t); got != message {
					t.Fatalf("连接 %s 收到的发布数据不符合预期：期望 %q，实际 %q", client.uniqId, message, got)
				}
			}
		}
	})
}

// TestTopicPublishBulk 批量发布：每一项是一组主题与其数据
func TestTopicPublishBulk(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		topics := []string{uniqueData("topic"), uniqueData("topic")}
		for _, uniqId := range uniqIds {
			e.bus.TopicSubscribe(uniqId, topics, nil)
		}
		// 每一项一个主题、一条数据
		dataList := make([]string, 0, len(topics))
		items := make([]*netsvrProtocol.TopicPublishBulkItem, 0, len(topics))
		for _, topic := range topics {
			message := uniqueData("topicPublishBulk")
			dataList = append(dataList, message)
			item := &netsvrProtocol.TopicPublishBulkItem{Topics: []string{topic}, Data: [][]byte{[]byte(message)}}
			items = append(items, item)
		}
		e.bus.TopicPublishBulk(items)
		for _, client := range e.allClients() {
			for _, want := range dataList {
				if got := client.receive(t); got != want {
					t.Fatalf("连接 %s 收到的批量发布数据不符合预期：期望 %q，实际 %q", client.uniqId, want, got)
				}
			}
		}
		// 一项内一个主题接收多条数据
		dataList = []string{uniqueData("topicPublishBulk"), uniqueData("topicPublishBulk")}
		item := &netsvrProtocol.TopicPublishBulkItem{Topics: []string{topics[0]}, Data: [][]byte{[]byte(dataList[0]), []byte(dataList[1])}}
		e.bus.TopicPublishBulk([]*netsvrProtocol.TopicPublishBulkItem{item})
		for _, client := range e.allClients() {
			for _, want := range dataList {
				if got := client.receive(t); got != want {
					t.Fatalf("连接 %s 收到的批量发布数据不符合预期：期望 %q，实际 %q", client.uniqId, want, got)
				}
			}
		}
	})
}

// TestConnInfoUpdateDelete 更新连接信息、查看、删除
func TestConnInfoUpdateDelete(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		topics := []string{uniqueData("topic")}
		e.updateConnInfo(uniqIds, func(uniqId string, up *netsvrProtocol.ConnInfoUpdate) {
			up.NewSession = uniqId + "Session"
			up.NewCustomerId = uniqId + "CustomerId"
			up.NewTopics = topics
		})
		ret := e.bus.ConnInfo(uniqIds, true, true, true)
		for _, uniqId := range uniqIds {
			item, ok := ret.Get(uniqId)
			if !ok {
				t.Fatalf("连接 %s 的信息不存在", uniqId)
			}
			if item.GetSession() != uniqId+"Session" {
				t.Fatalf("连接 %s 的 session 不符合预期：%q", uniqId, item.GetSession())
			}
			if item.GetCustomerId() != uniqId+"CustomerId" {
				t.Fatalf("连接 %s 的 customerId 不符合预期：%q", uniqId, item.GetCustomerId())
			}
			assertSameSet(t, "连接的主题", topics, item.GetTopics())
		}
		// 删除连接上存储的信息
		for _, uniqId := range uniqIds {
			del := &netsvrProtocol.ConnInfoDelete{UniqId: uniqId, DelSession: true, DelCustomerId: true, DelTopic: true}
			e.bus.ConnInfoDelete(del)
		}
		ret = e.bus.ConnInfo(uniqIds, true, true, true)
		for _, uniqId := range uniqIds {
			item, ok := ret.Get(uniqId)
			if !ok {
				t.Fatalf("连接 %s 的信息不存在", uniqId)
			}
			if item.GetSession() != "" || item.GetCustomerId() != "" || len(item.GetTopics()) != 0 {
				t.Fatalf("连接 %s 的信息删除不干净：session=%q customerId=%q topics=%v", uniqId, item.GetSession(), item.GetCustomerId(), item.GetTopics())
			}
		}
	})
}

// TestForceOffline 强制关闭连接：先收到转发的数据，再收到关闭帧
func TestForceOffline(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		message := uniqueData("forceOffline")
		e.bus.ForceOffline(uniqIds, []byte(message))
		for _, client := range e.allClients() {
			if got := client.receive(t); got != message {
				t.Fatalf("连接 %s 收到的下线数据不符合预期：期望 %q，实际 %q", client.uniqId, message, got)
			}
			if code := client.receiveClose(t); code != forceOfflineCloseCode {
				t.Fatalf("连接 %s 的关闭码不符合预期：期望 %d，实际 %d", client.uniqId, forceOfflineCloseCode, code)
			}
		}
		e.waitOffline(uniqIds)
	})
}

// TestForceOfflineByCustomerId 按 customerId 强制关闭某个客户的所有连接
func TestForceOfflineByCustomerId(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		// 每个连接使用独立的 customerId
		e.updateConnInfo(uniqIds, func(uniqId string, up *netsvrProtocol.ConnInfoUpdate) {
			up.NewCustomerId = uniqId
		})
		e.bus.ForceOfflineByCustomerId(uniqIds, nil)
		for _, client := range e.allClients() {
			if code := client.receiveClose(t); code != forceOfflineCloseCode {
				t.Fatalf("连接 %s 的关闭码不符合预期：期望 %d，实际 %d", client.uniqId, forceOfflineCloseCode, code)
			}
		}
		e.waitOffline(uniqIds)
	})
}

// TestForceOfflineGuest 只强制关闭 session 为空的连接
func TestForceOfflineGuest(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		// session 为空，会被关闭
		e.bus.ForceOfflineGuest(uniqIds, nil, 0)
		for _, client := range e.allClients() {
			if code := client.receiveClose(t); code != forceOfflineCloseCode {
				t.Fatalf("连接 %s 的关闭码不符合预期：期望 %d，实际 %d", client.uniqId, forceOfflineCloseCode, code)
			}
		}
		e.waitOffline(uniqIds)
		// session 不为空，不会被关闭
		otherEnv := newEnv(t, gateways)
		otherUniqIds := otherEnv.uniqIds()
		otherEnv.updateConnInfo(otherUniqIds, func(uniqId string, up *netsvrProtocol.ConnInfoUpdate) {
			up.NewSession = uniqId
		})
		otherEnv.bus.ForceOfflineGuest(otherUniqIds, nil, 0)
		if ret := otherEnv.bus.CheckOnline(otherUniqIds); len(ret.UniqIds()) != len(otherUniqIds) {
			t.Fatalf("有 session 的连接不应被强制下线：期望 %d 个在线，实际 %v", len(otherUniqIds), ret.UniqIds())
		}
	})
}

// TestSingleCastBulkSkipInvalidTarget 目标不存在、目标为空、数据为空时网关会跳过，且不影响同项内的其它目标
func TestSingleCastBulkSkipInvalidTarget(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		// uniqId 的前 12 个十六进制字符是网关地址，尾部换成不存在的自增 id，
		// 构造一个「落在同一个网关、但连接不存在」的目标
		notExistUniqId := uniqIds[0][:12] + "ffffffffffffffff"
		validFirst := uniqIds[0]
		validSecond := uniqIds[1]
		validThird := uniqIds[2]
		dataFirst := uniqueData("skipInvalidTarget")
		dataSecond := uniqueData("skipInvalidTarget")
		items := []*netsvrProtocol.SingleCastBulkItem{
			// 不存在的目标与真实目标混在同一项：真实目标照常收到数据
			{UniqIds: []string{notExistUniqId, validFirst}, Data: [][]byte{[]byte(dataFirst)}},
			// 同一项内混入空数据：空数据被跳过，真实数据照常投递
			{UniqIds: []string{validSecond}, Data: [][]byte{[]byte(dataSecond), {}}},
			// 目标为空：整项不处理
			{UniqIds: nil, Data: [][]byte{[]byte(uniqueData("skipInvalidTarget"))}},
			// 数据为空：整项不处理
			{UniqIds: []string{validThird}, Data: nil},
		}
		e.bus.SingleCastBulk(items)
		if got := e.clientOf(validFirst).receive(t); got != dataFirst {
			t.Fatalf("连接 %s 收到的数据不符合预期：期望 %q，实际 %q", validFirst, dataFirst, got)
		}
		if got := e.clientOf(validSecond).receive(t); got != dataSecond {
			t.Fatalf("连接 %s 收到的数据不符合预期：期望 %q，实际 %q", validSecond, dataSecond, got)
		}
		// 目标为空、数据为空的项不应投递任何数据，不存在的目标也不影响其它连接
		e.clientOf(validThird).assertNoMessage(t, 300*time.Millisecond)
		for _, client := range e.allClients() {
			if client.uniqId == validFirst || client.uniqId == validSecond {
				continue
			}
			client.assertNoMessage(t, 300*time.Millisecond)
		}
	})
}

// TestForceOfflineUncooperativeClient 客户端收到关闭帧后既不回关闭帧、也不断开连接，
// 网关仍会在兜底时间到达后关闭连接（不会把连接一直挂着）
func TestForceOfflineUncooperativeClient(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		// 收到关闭帧时不回关闭帧，也不主动断开 TCP，让网关走 2 秒兜底关闭的分支
		for _, client := range e.allClients() {
			client.conn.SetCloseHandler(func(code int, text string) error {
				return nil
			})
		}
		e.bus.ForceOffline(uniqIds, nil)
		for _, client := range e.allClients() {
			if code := client.receiveClose(t); code != forceOfflineCloseCode {
				t.Fatalf("连接 %s 的关闭码不符合预期：期望 %d，实际 %d", client.uniqId, forceOfflineCloseCode, code)
			}
		}
		// 兜底关闭到达后，连接会从网关的在线列表里消失
		e.waitOfflineWithin(uniqIds, 5*time.Second)
	})
}

// TestSendToSharedCustomerId 同一个客户连接到多个网关时，给该客户发数据，各网关上的连接都能收到
func TestSendToSharedCustomerId(t *testing.T) {
	forEachDeployment(t, func(t *testing.T, gateways []gatewayConfig) {
		e := newEnv(t, gateways)
		uniqIds := e.uniqIds()
		e.setUniqueCustomerId(uniqIds)
		sharedCustomerId, sharedUniqIds := e.shareCustomerId(uniqIds)
		shared := make(map[string]struct{}, len(sharedUniqIds))
		for _, uniqId := range sharedUniqIds {
			shared[uniqId] = struct{}{}
		}
		// 组播：该客户的全部连接（含跨网关）都收到
		message := uniqueData("sendToSharedCustomerId")
		e.bus.SendToCustomerIds([]string{sharedCustomerId}, []byte(message))
		for _, uniqId := range sharedUniqIds {
			if got := e.clientOf(uniqId).receive(t); got != message {
				t.Fatalf("连接 %s 收到的数据不符合预期：期望 %q，实际 %q", uniqId, message, got)
			}
		}
		// 批量单播：同上
		message = uniqueData("singleCastBulkBySharedCustomerId")
		e.bus.SingleCastBulkByCustomerId([]*netsvrProtocol.SingleCastBulkByCustomerIdItem{
			{CustomerIds: []string{sharedCustomerId}, Data: [][]byte{[]byte(message)}},
		})
		for _, uniqId := range sharedUniqIds {
			if got := e.clientOf(uniqId).receive(t); got != message {
				t.Fatalf("连接 %s 收到的数据不符合预期：期望 %q，实际 %q", uniqId, message, got)
			}
		}
		// 其它客户不应收到数据
		for _, client := range e.allClients() {
			if _, ok := shared[client.uniqId]; ok {
				continue
			}
			client.assertNoMessage(t, 300*time.Millisecond)
		}
	})
}
