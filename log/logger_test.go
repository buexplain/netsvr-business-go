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

package log

import (
	"strings"
	"testing"
)

// recordLogger 记录日志调用的测试实现
type recordLogger struct {
	records []string
}

func (r *recordLogger) Debug(msg string, args ...any) {
	r.records = append(r.records, "debug:"+msg)
}

func (r *recordLogger) Info(msg string, args ...any) {
	r.records = append(r.records, "info:"+msg)
}

func (r *recordLogger) Warn(msg string, args ...any) {
	r.records = append(r.records, "warn:"+msg)
}

func (r *recordLogger) Error(msg string, args ...any) {
	r.records = append(r.records, "error:"+msg)
}

func TestLogger(t *testing.T) {
	defaultLogger := GetLogger()
	if defaultLogger == nil {
		t.Fatalf("默认日志实现不应为空")
	}
	recorder := &recordLogger{}
	SetLogger(recorder)
	t.Cleanup(func() {
		SetLogger(defaultLogger)
	})
	if GetLogger() != recorder {
		t.Fatalf("SetLogger 未生效")
	}
	Debug("d")
	Info("i")
	Warn("w")
	Error("e")
	if got := strings.Join(recorder.records, ","); got != "debug:d,info:i,warn:w,error:e" {
		t.Fatalf("日志调用不符合预期：%s", got)
	}
}
