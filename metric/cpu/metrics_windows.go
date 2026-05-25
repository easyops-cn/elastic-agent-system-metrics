// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

/*
For testing via the win2012 vagrant box:
vagrant winrm -s cmd -e -c "cd C:\\Gopath\src\\github.com\\elastic\\beats\\metricbeat\\module\\system\\cpu; go test -v -tags=integration -run TestFetch"  win2012
*/

package cpu

import (
	"fmt"
	"time"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
	"github.com/elastic/gosigar/sys/windows"
)

// Get fetches Windows CPU system times
func Get(_ resolve.Resolver) (CPUMetrics, error) {
	idle, kernel, user, err := windows.GetSystemTimes()
	if err != nil {
		return CPUMetrics{}, fmt.Errorf("call to GetSystemTimes failed: %w", err)
	}

	globalMetrics := CPUMetrics{}

	// Use int64 intermediate variables to avoid uint64 overflow when
	// gosigar returns negative kernel (happens when rawIdle > rawKernel
	// on some 64-core Windows machines).
	totalMs := int64(idle/time.Millisecond) + int64(kernel/time.Millisecond) + int64(user/time.Millisecond)
	if totalMs < 0 {
		totalMs = 0
	}

	// convert from duration to ticks
	idleMetric := uint64(idle / time.Millisecond)
	sysMetric := opt.Uint{}
	if kernel >= 0 {
		sysMetric = opt.UintWith(uint64(kernel / time.Millisecond))
	}
	userMetrics := uint64(user / time.Millisecond)
	globalMetrics.totals.Idle = opt.UintWith(idleMetric)
	globalMetrics.totals.Sys = sysMetric
	globalMetrics.totals.User = opt.UintWith(userMetrics)
	globalMetrics.totals.TotalTicks = opt.UintWith(uint64(totalMs))

	// get per-cpu data
	cpus, err := windows.NtQuerySystemProcessorPerformanceInformation()
	if err != nil {
		return CPUMetrics{}, fmt.Errorf("catll to NtQuerySystemProcessorPerformanceInformation failed: %w", err)
	}
	globalMetrics.list = make([]CPU, 0, len(cpus))
	for _, cpu := range cpus {
		perIdleMs := int64(cpu.IdleTime / time.Millisecond)
		perKernelMs := int64(cpu.KernelTime / time.Millisecond)
		perUserMs := int64(cpu.UserTime / time.Millisecond)
		perTotalMs := perIdleMs + perKernelMs + perUserMs
		if perTotalMs < 0 {
			perTotalMs = 0
		}
		perSysMs := opt.Uint{}
		if perKernelMs >= 0 {
			perSysMs = opt.UintWith(uint64(perKernelMs))
		}

		globalMetrics.list = append(globalMetrics.list, CPU{
			Idle:       opt.UintWith(uint64(perIdleMs)),
			Sys:        perSysMs,
			User:       opt.UintWith(uint64(perUserMs)),
			TotalTicks: opt.UintWith(uint64(perTotalMs)),
		})
	}

	return globalMetrics, nil
}
