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
	"errors"
	"fmt"
	"net"
	"slices"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buexplain/netsvr-business-go/v3/taskSocket"
	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
	"github.com/gorilla/websocket"
)

const (
	// 网关的 customer、worker、task 三个服务配置的心跳字符串一致
	taskHeartbeatMessage = "~6YOt5rW35piO~"
	// 每个网关上建立的测试连接数
	onlineNumPerGateway = 3
)

// gatewayConfig 一个网关实例对外提供的三个服务地址
type gatewayConfig struct {
	// task 服务地址，NetBus 连它
	taskAddr string
	// worker 服务地址，mainSocket 连它
	workerAddr string
	// customer（websocket）服务地址，测试客户端连它
	wsURL string
}

// deployment 一种部署形态
type deployment struct {
	name     string
	gateways []gatewayConfig
}

// deployments 返回单机与分布式两种部署形态，对应测试环境里的 607、608 两个网关实例
func deployments() []deployment {
	return []deployment{
		{
			name: "single",
			gateways: []gatewayConfig{
				{taskAddr: "127.0.0.1:6072", workerAddr: "127.0.0.1:6071", wsURL: "ws://127.0.0.1:6070/netsvr"},
			},
		},
		{
			name: "distributed",
			gateways: []gatewayConfig{
				{taskAddr: "127.0.0.1:6072", workerAddr: "127.0.0.1:6071", wsURL: "ws://127.0.0.1:6070/netsvr"},
				{taskAddr: "127.0.0.1:6082", workerAddr: "127.0.0.1:6081", wsURL: "ws://127.0.0.1:6080/netsvr"},
			},
		},
	}
}

// forEachDeployment 让用例在两种部署形态下各跑一遍；网关没启动则跳过
func forEachDeployment(t *testing.T, f func(t *testing.T, gateways []gatewayConfig)) {
	t.Helper()
	for _, d := range deployments() {
		t.Run(d.name, func(t *testing.T) {
			skipUnlessGatewayReady(t, d.gateways)
			f(t, d.gateways)
		})
	}
}

// skipUnlessGatewayReady 集成测试依赖正在运行的网关，连不上时跳过而不是报失败
func skipUnlessGatewayReady(t *testing.T, gateways []gatewayConfig) {
	t.Helper()
	for _, gw := range gateways {
		for _, addr := range []string{gw.workerAddr, gw.taskAddr} {
			conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
			if err != nil {
				t.Skipf("网关未运行（%s 不可达：%v），跳过集成测试", addr, err)
			}
			_ = conn.Close()
		}
	}
}

var dataSequence int64

// uniqueData 生成唯一的测试数据，避免用例之间相互干扰
func uniqueData(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, atomic.AddInt64(&dataSequence, 1))
}

// wsClient 测试用的 websocket 客户端，模拟一个真实的客户连接
type wsClient struct {
	conn   *websocket.Conn
	uniqId string
}

// newWsClient 连接网关的 customer 服务，并读取网关下发的 uniqId
func newWsClient(t *testing.T, wsURL string) *wsClient {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("连接网关 %s 失败：%v", wsURL, err)
	}
	client := &wsClient{conn: conn}
	t.Cleanup(client.close)
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("设置读超时失败：%v", err)
	}
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("读取网关下发的 uniqId 失败：%v", err)
	}
	client.uniqId = string(message)
	return client
}

// receive 读取一条数据，超时视为失败
func (c *wsClient) receive(t *testing.T) string {
	t.Helper()
	if err := c.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("设置读超时失败：%v", err)
	}
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		t.Fatalf("连接 %s 读取数据失败：%v", c.uniqId, err)
	}
	return string(message)
}

// receiveClose 读取关闭帧，返回关闭码
func (c *wsClient) receiveClose(t *testing.T) int {
	t.Helper()
	if err := c.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("设置读超时失败：%v", err)
	}
	_, _, err := c.conn.ReadMessage()
	var closeErr *websocket.CloseError
	if !errors.As(err, &closeErr) {
		t.Fatalf("连接 %s 未收到关闭帧：%v", c.uniqId, err)
	}
	return closeErr.Code
}

// assertNoMessage 断言连接在给定时间内收不到数据
func (c *wsClient) assertNoMessage(t *testing.T, timeout time.Duration) {
	t.Helper()
	if err := c.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("设置读超时失败：%v", err)
	}
	_, message, err := c.conn.ReadMessage()
	var netErr net.Error
	if err == nil || !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("连接 %s 不应收到数据，实际收到 %q（err=%v）", c.uniqId, message, err)
	}
}

// close 关闭连接
func (c *wsClient) close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// env 一次集成测试的运行环境：NetBus 与每个网关上的真实 websocket 连接
type env struct {
	t        *testing.T
	gateways []gatewayConfig
	bus      *NetBus
	// key 是网关的 task 地址
	clients map[string][]*wsClient
}

// newEnv 建立到各网关的 task 连接池与测试用 websocket 连接
func newEnv(t *testing.T, gateways []gatewayConfig) *env {
	t.Helper()
	manger := taskSocket.NewManger()
	for _, gw := range gateways {
		factory := taskSocket.NewFactory(gw.taskAddr, 5*time.Second, 5*time.Second, 5*time.Second)
		pool := taskSocket.NewPool(onlineNumPerGateway, factory, 5*time.Second, 45*time.Second, []byte(taskHeartbeatMessage))
		pool.LoopHeartbeat()
		manger.AddSocket(pool)
	}
	e := &env{
		t:        t,
		gateways: gateways,
		bus:      NewNetBus(manger),
		clients:  make(map[string][]*wsClient, len(gateways)),
	}
	t.Cleanup(e.bus.Close)
	for _, gw := range gateways {
		for i := 0; i < onlineNumPerGateway; i++ {
			e.clients[gw.taskAddr] = append(e.clients[gw.taskAddr], newWsClient(t, gw.wsURL))
		}
	}
	return e
}

// uniqIds 返回全部连接（跨网关）的 uniqId
func (e *env) uniqIds() []string {
	ret := make([]string, 0, len(e.gateways)*onlineNumPerGateway)
	for _, gw := range e.gateways {
		ret = append(ret, e.uniqIdsByTaskAddr(gw.taskAddr)...)
	}
	return ret
}

// uniqIdsByTaskAddr 返回某个网关上的 uniqId
func (e *env) uniqIdsByTaskAddr(taskAddr string) []string {
	clients := e.clients[taskAddr]
	ret := make([]string, 0, len(clients))
	for _, client := range clients {
		ret = append(ret, client.uniqId)
	}
	return ret
}

// allClients 返回全部连接
func (e *env) allClients() []*wsClient {
	ret := make([]*wsClient, 0, len(e.gateways)*onlineNumPerGateway)
	for _, gw := range e.gateways {
		ret = append(ret, e.clients[gw.taskAddr]...)
	}
	return ret
}

// clientOf 按 uniqId 找到对应的测试连接
func (e *env) clientOf(uniqId string) *wsClient {
	for _, gw := range e.gateways {
		for _, client := range e.clients[gw.taskAddr] {
			if client.uniqId == uniqId {
				return client
			}
		}
	}
	e.t.Fatalf("找不到 uniqId=%s 的测试连接", uniqId)
	return nil
}

// updateConnInfo 给每个连接设置信息，mutate 负责填写要设置的字段
func (e *env) updateConnInfo(uniqIds []string, mutate func(uniqId string, up *netsvrProtocol.ConnInfoUpdate)) {
	e.t.Helper()
	for _, uniqId := range uniqIds {
		up := &netsvrProtocol.ConnInfoUpdate{UniqId: uniqId}
		mutate(uniqId, up)
		e.bus.ConnInfoUpdate(up)
	}
}

// waitOffline 等待这些连接从网关下线；强制关闭是异步的，网关写完关闭帧后还要清理连接
func (e *env) waitOffline(uniqIds []string) {
	e.t.Helper()
	e.waitOfflineWithin(uniqIds, 3*time.Second)
}

// waitOfflineWithin 在给定时间内等待这些连接从网关下线
func (e *env) waitOfflineWithin(uniqIds []string, timeout time.Duration) {
	e.t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if len(e.bus.CheckOnline(uniqIds).UniqIds()) == 0 {
			return
		}
		if time.Now().After(deadline) {
			e.t.Fatalf("等待连接下线超时，仍在线：%v", e.bus.CheckOnline(uniqIds).UniqIds())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// 关闭码 1008：网关强制下线连接时写入的关闭码
const forceOfflineCloseCode = 1008

// sortedUnique 去重并排序，便于忽略顺序地比较集合
func sortedUnique(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	ret := make([]string, 0, len(in))
	for _, item := range in {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		ret = append(ret, item)
	}
	sort.Strings(ret)
	return ret
}

// assertSameSet 忽略顺序与重复地比较两个字符串集合
func assertSameSet(t *testing.T, msg string, want, got []string) {
	t.Helper()
	wantSorted := sortedUnique(want)
	gotSorted := sortedUnique(got)
	if len(wantSorted) != len(gotSorted) {
		t.Fatalf("%s：期望 %v，实际 %v", msg, want, got)
	}
	for i := range wantSorted {
		if wantSorted[i] != gotSorted[i] {
			t.Fatalf("%s：期望 %v，实际 %v", msg, want, got)
		}
	}
}

// assertSameList 忽略顺序但**不忽略重复**地比较两个列表。
// 断言「跨网关去重」「合并后不应有重复项」这类语义时必须用它：
// assertSameSet 会先去掉重复，结果里混进重复项时它看不出来。
func assertSameList(t *testing.T, msg string, want, got []string) {
	t.Helper()
	wantSorted := slices.Clone(want)
	gotSorted := slices.Clone(got)
	slices.Sort(wantSorted)
	slices.Sort(gotSorted)
	if !slices.Equal(wantSorted, gotSorted) {
		t.Fatalf("%s：期望 %v，实际 %v", msg, want, got)
	}
}
