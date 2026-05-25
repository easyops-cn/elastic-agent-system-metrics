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

package cpu

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

func TestMonitorSample(t *testing.T) {
	cpu := &Monitor{lastSample: CPUMetrics{}, Hostfs: resolve.NewTestResolver("")}
	s, err := cpu.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	metricOpts := MetricOpts{Percentages: true, NormalizedPercentages: true, Ticks: true}
	evt, err := s.Format(metricOpts)
	assert.NoError(t, err, "error in Format")
	testPopulatedEvent(evt, t, true)
}

func TestCoresMonitorSample(t *testing.T) {

	cpuMetrics, err := Get(resolve.NewTestResolver(""))
	assert.NoError(t, err, "error in Get()")

	cores := &Monitor{lastSample: CPUMetrics{list: make([]CPU, len(cpuMetrics.list))}, Hostfs: resolve.NewTestResolver("")}
	sample, err := cores.FetchCores()
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range sample {
		metricOpts := MetricOpts{Percentages: true, Ticks: true}
		evt, err := s.Format(metricOpts)
		assert.NoError(t, err, "error in Format")
		testPopulatedEvent(evt, t, false)
	}
}

func testPopulatedEvent(evt mapstr.M, t *testing.T, norm bool) {
	user, err := evt.GetValue("user.pct")
	assert.NoError(t, err, "error getting user.pct")
	system, err := evt.GetValue("system.pct")
	assert.NoError(t, err, "error getting system.pct")
	assert.True(t, user.(float64) > 0)
	assert.True(t, system.(float64) > 0)

	if norm {
		normUser, err := evt.GetValue("user.norm.pct")
		assert.NoError(t, err, "error getting user.norm.pct")
		assert.True(t, normUser.(float64) > 0)
		normSystem, err := evt.GetValue("system.norm.pct")
		assert.NoError(t, err, "error getting system.norm.pct")
		assert.True(t, normSystem.(float64) > 0)
		assert.True(t, normUser.(float64) <= 100)
		assert.True(t, normSystem.(float64) <= 100)

		assert.True(t, user.(float64) > normUser.(float64))
		assert.True(t, system.(float64) > normSystem.(float64))
	}

	userTicks, err := evt.GetValue("user.ticks")
	assert.NoError(t, err, "error getting user.ticks")
	assert.True(t, userTicks.(uint64) > 0)
	systemTicks, err := evt.GetValue("system.ticks")
	assert.NoError(t, err, "error getting system.ticks")
	assert.True(t, systemTicks.(uint64) > 0)
}

// TestMetricsRounding tests that the returned percentages are rounded to
// four decimal places.
func TestMetricsRounding(t *testing.T) {

	sample := Metrics{
		previousSample: CPU{
			User: opt.UintWith(10855311),
			Sys:  opt.UintWith(2021040),
			Idle: opt.UintWith(17657874),
		},
		currentSample: CPU{
			User: opt.UintWith(10855693),
			Sys:  opt.UintWith(2021058),
			Idle: opt.UintWith(17657876),
		},
	}

	evt, err := sample.Format(MetricOpts{NormalizedPercentages: true})
	assert.NoError(t, err, "error in Format")
	normUser, err := evt.GetValue("user.norm.pct")
	assert.NoError(t, err, "error getting user.norm.pct")
	normSystem, err := evt.GetValue("system.norm.pct")
	assert.NoError(t, err, "error getting system.norm.pct")

	assert.Equal(t, normUser.(float64), 0.9502)
	assert.Equal(t, normSystem.(float64), 0.0448)
}

// TestMetricsPercentages tests that Metrics returns the correct
// percentages and normalized percentages.
func TestMetricsPercentages(t *testing.T) {
	numCores := 10
	// This test simulates 30% user and 70% system (normalized), or 3% and 7%
	// respectively when there are 10 CPUs.
	const userTest, systemTest = 30., 70.

	s0 := CPU{
		User: opt.UintWith(10000000),
		Sys:  opt.UintWith(10000000),
		Idle: opt.UintWith(20000000),
		Nice: opt.UintWith(0),
	}
	s1 := CPU{
		User: opt.UintWith(s0.User.ValueOr(0) + uint64(userTest)),
		Sys:  opt.UintWith(s0.Sys.ValueOr(0) + uint64(systemTest)),
		Idle: s0.Idle,
		Nice: opt.UintWith(0),
	}
	sample := Metrics{
		count:          numCores,
		isTotals:       true,
		previousSample: s0,
		currentSample:  s1,
	}

	evt, err := sample.Format(MetricOpts{NormalizedPercentages: true, Percentages: true})
	assert.NoError(t, err, "error in Format")

	user, err := evt.GetValue("user.norm.pct")
	assert.NoError(t, err, "error getting user.norm.pct")
	system, err := evt.GetValue("system.norm.pct")
	assert.NoError(t, err, "error getting system.norm.pct")
	idle, err := evt.GetValue("idle.norm.pct")
	assert.NoError(t, err, "error getting idle.norm.pct")
	total, err := evt.GetValue("total.norm.pct")
	assert.NoError(t, err, "error getting total.norm.pct")
	assert.EqualValues(t, .3, user.(float64))
	assert.EqualValues(t, .7, system.(float64))
	assert.EqualValues(t, .0, idle.(float64))
	assert.EqualValues(t, 1., total.(float64))
}

// TestTotalTicksPreferred tests that Total() returns the pre-computed
// TotalTicks value when set, rather than summing the individual fields.
func TestTotalTicksPreferred(t *testing.T) {
	cpu := CPU{
		User:       opt.UintWith(100),
		Sys:        opt.UintWith(0),
		Idle:       opt.UintWith(9000),
		TotalTicks: opt.UintWith(10000),
	}
	assert.Equal(t, uint64(10000), cpu.Total())

	// Without TotalTicks, Total falls back to summing fields
	cpu2 := CPU{
		User:       opt.UintWith(100),
		Sys:        opt.UintWith(0),
		Idle:       opt.UintWith(9000),
	}
	assert.Equal(t, uint64(9100), cpu2.Total())
}

// TestWindowsNegativeKernelDerivation tests that when Sys is zero-value
// (Windows kernel < idle overflow, so Sys was left as opt.Uint{}),
// system pct is derived as totalPct - userPct, and all percentages are valid.
func TestWindowsNegativeKernelDerivation(t *testing.T) {
	// Layout: idle=70%, user=20%, system=10% of total.
	// Sys is not set (IsZero()=true) to simulate Windows overflow path.
	// TotalTicks carries the correct total computed via int64.
	prev := CPU{
		User:       opt.UintWith(20000),
		Idle:       opt.UintWith(70000),
		Sys:        opt.Uint{}, // overflowed, not set
		TotalTicks: opt.UintWith(100000),
	}
	cur := CPU{
		User:       opt.UintWith(22000),  // +2000
		Idle:       opt.UintWith(77000),  // +7000
		Sys:        opt.Uint{},           // still overflowed
		TotalTicks: opt.UintWith(110000), // +10000
	}

	sample := Metrics{
		count:          64,
		isTotals:       true,
		previousSample: prev,
		currentSample:  cur,
	}

	evt, err := sample.Format(MetricOpts{NormalizedPercentages: true, Percentages: true})
	assert.NoError(t, err)

	// total.norm.pct = 1 - idlePct = 1 - 0.7 = 0.30
	totalNormPct, _ := evt.GetValue("total.norm.pct")
	assert.InDelta(t, 0.30, totalNormPct.(float64), 0.01)

	// user.norm.pct = 2000/10000 = 0.20
	userNormPct, _ := evt.GetValue("user.norm.pct")
	assert.InDelta(t, 0.20, userNormPct.(float64), 0.01)

	// system.norm.pct = totalPct - userPct = 0.30 - 0.20 = 0.10
	sysNormPct, _ := evt.GetValue("system.norm.pct")
	assert.InDelta(t, 0.10, sysNormPct.(float64), 0.01)

	// idle + user + system ≈ 1.0 (normalized)
	idleNormPct, _ := evt.GetValue("idle.norm.pct")
	sum := idleNormPct.(float64) + userNormPct.(float64) + sysNormPct.(float64)
	assert.InDelta(t, 1.0, sum, 0.01, "idle+user+system should sum to ~1.0 (normalized)")
}
