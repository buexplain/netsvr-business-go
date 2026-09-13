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

package taskSocket

import (
	"bytes"
	"github.com/buexplain/netsvr-business-go/v3/log"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// gatewayTaskAddr 测试环境里网关的 task 服务地址
const gatewayTaskAddr = "127.0.0.1:6072"

// skipUnlessGatewayReady 集成测试依赖正在运行的网关，连不上时跳过而不是失败
func skipUnlessGatewayReady(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", gatewayTaskAddr, 500*time.Millisecond)
	if err != nil {
		t.Skipf("网关未运行（%s 不可达：%v），跳过集成测试", gatewayTaskAddr, err)
	}
	_ = conn.Close()
}

func TestTaskSocketPool_NewPool(t *testing.T) {
	factory := NewFactory(gatewayTaskAddr, time.Second*10, time.Second*10, time.Second*10)
	pool := NewPool(10, factory, time.Second*10, time.Second*10, []byte("~6YOt5rW35piO~"))
	defer pool.Close()
	if pool.pool == nil || cap(pool.pool) != 10 {
		t.Error("NewPool failed")
	}
	if pool.size == nil || cap(pool.size) != 10 {
		t.Error("NewPool failed")
	}
	if pool.factory != factory {
		t.Error("NewPool failed")
	}
	if pool.waitTimeout != time.Second*10 {
		t.Error("NewPool failed")
	}
}

func TestTaskSocketPool_GetAddr(t *testing.T) {
	factory := NewFactory(gatewayTaskAddr, time.Second*10, time.Second*10, time.Second*10)
	pool := NewPool(10, factory, time.Second*10, time.Second*10, []byte("~6YOt5rW35piO~"))
	defer pool.Close()
	if pool.GetAddr() != factory.GetAddr() {
		t.Error("GetAddr failed")
	}
}

func TestTaskSocketPool_Get(t *testing.T) {
	skipUnlessGatewayReady(t)
	size := 10
	factory := NewFactory(gatewayTaskAddr, time.Second*10, time.Second*10, time.Second*10)
	pool := NewPool(size, factory, time.Second*10, time.Second*10, []byte("~6YOt5rW35piO~"))
	defer pool.Close()
	if pool.GetAddr() != factory.GetAddr() {
		t.Error("GetAddr failed")
	}
	taskSocket := pool.Get()
	if taskSocket == nil {
		t.Error("Get failed")
		return
	}
	defer taskSocket.Release()
	if taskSocket.GetAddr() != factory.GetAddr() {
		t.Error("GetAddr failed")
	}
	if taskSocket.IsConnected() == false {
		t.Error("Get failed")
	}
	if len(pool.size) != size-1 {
		t.Error("Get failed")
	}
	if len(pool.pool) > 0 {
		t.Error("Get failed")
	}
}

func TestTaskSocketPool_ConcurrencyGet(t *testing.T) {
	skipUnlessGatewayReady(t)
	size := 10
	factory := NewFactory(gatewayTaskAddr, time.Second*10, time.Second*10, time.Second*10)
	pool := NewPool(size, factory, time.Second*10, time.Second*10, []byte("~6YOt5rW35piO~"))
	defer pool.Close()
	wg := &sync.WaitGroup{}
	for i := size * 2; i > 0; i-- {
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
			}()
			taskSocket := pool.Get()
			if taskSocket != nil {
				<-time.After(time.Second)
				taskSocket.Release()
			}
		}()
	}
	wg.Wait()
	if len(pool.size) > 0 {
		t.Error("ConcurrencyGet failed")
	}
	if len(pool.pool) != size {
		t.Error("ConcurrencyGet failed")
	}
}

func TestTaskSocketPool_WaitTimeoutGet(t *testing.T) {
	skipUnlessGatewayReady(t)
	size := 2
	factory := NewFactory(gatewayTaskAddr, time.Second*10, time.Second*10, time.Second*10)
	pool := NewPool(size, factory, time.Second*10, time.Second*10, []byte("~6YOt5rW35piO~"))
	defer pool.Close()
	taskSocketList := make([]*TaskSocket, 0, size)
	for i := 0; i < size+1; i++ {
		taskSocket := pool.Get()
		if i == size {
			if taskSocket != nil {
				t.Error("WaitTimeoutGet failed")
			}
			continue
		}
		if taskSocket == nil {
			t.Fatalf("获取 task 连接失败")
		}
		taskSocketList = append(taskSocketList, taskSocket)
	}
	if len(pool.size) > 0 {
		t.Error("WaitTimeoutGet failed")
	}
	if len(pool.pool) > 0 {
		t.Error("WaitTimeoutGet failed")
	}
	for _, taskSocket := range taskSocketList {
		taskSocket.Release()
	}
	if len(pool.size) > 0 {
		t.Error("WaitTimeoutGet failed")
	}
	if len(pool.pool) != size {
		t.Error("WaitTimeoutGet failed")
	}
}

func TestTaskSocketPool_LoopHeartbeat(t *testing.T) {
	skipUnlessGatewayReady(t)
	stdOut := bytes.NewBuffer(nil)
	defaultLog := log.GetLogger()
	log.SetLogger(slog.New(slog.NewTextHandler(stdOut, nil)))
	defer func() {
		log.SetLogger(defaultLog)
	}()
	size := 10
	factory := NewFactory(gatewayTaskAddr, time.Second*10, time.Second*10, time.Second*10)
	pool := NewPool(size, factory, time.Second*10, time.Millisecond*100, []byte("~6YOt5rW35piO~"))
	taskSocketList := make([]*TaskSocket, 0, size)
	for i := 0; i < size; i++ {
		taskSocket := pool.Get()
		if taskSocket == nil {
			t.Fatalf("获取 task 连接失败")
		}
		taskSocketList = append(taskSocketList, taskSocket)
	}
	for _, taskSocket := range taskSocketList {
		taskSocket.Release()
	}
	pool.LoopHeartbeat()
	time.Sleep(time.Second * 3)
	pool.Close()
	time.Sleep(time.Second * 1)
	logStr := stdOut.String()
	if !strings.Contains(logStr, "taskSocketPool loopHeartbeat "+factory.GetAddr()+" quit") {
		t.Error("LoopHeartbeat failed")
	}
}

func TestTaskSocketPool_Close(t *testing.T) {
	skipUnlessGatewayReady(t)
	size := 10
	factory := NewFactory(gatewayTaskAddr, time.Second*10, time.Second*10, time.Second*10)
	pool := NewPool(size, factory, time.Second*10, time.Second*10, []byte("~6YOt5rW35piO~"))
	taskSocketList := make([]*TaskSocket, 0, size)
	for i := 0; i < size; i++ {
		taskSocket := pool.Get()
		if taskSocket == nil {
			t.Fatalf("获取 task 连接失败")
		}
		taskSocketList = append(taskSocketList, taskSocket)
	}
	if len(pool.pool) != 0 {
		t.Error("Close failed")
	}
	if len(pool.size) > 0 {
		t.Error("Close failed")
	}
	for _, taskSocket := range taskSocketList {
		taskSocket.Release()
	}
	if len(pool.size) > 0 {
		t.Error("Close failed")
	}
	if len(pool.pool) != size {
		t.Error("Close failed")
	}
	pool.Close()
	if len(pool.size) != size {
		t.Error("Close failed")
	}
	if len(pool.pool) > 0 {
		t.Error("Close failed")
	}
}
