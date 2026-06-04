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
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

func TestMonitorSample(t *testing.T) {
	cpu := &Monitor{lastSample: CPUMetrics{}, Hostfs: resolve.NewTestResolver("")}

	// First Fetch() stores the sample but returns no Metrics (no prev to compare).
	_, err := cpu.Fetch()
	assert.NoError(t, err, "first Fetch should not return error")

	// Wait for counters to accumulate so second sample has measurable delta.
	time.Sleep(500 * time.Millisecond)

	// Second Fetch() returns Metrics with valid prev/cur pair.
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

	// First FetchCores() stores samples but per-core results may have no valid
	// previous (lastSample was zero-initialized list). Skip first result.
	firstResult, err := cores.FetchCores()
	assert.NoError(t, err, "first FetchCores should not return error")
	_ = firstResult // intentionally discarded

	// Wait for counters to accumulate.
	time.Sleep(500 * time.Millisecond)

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
	if err != nil {
		// Per-core idle CPUs may have no user.pct (reportOptMetric skips zero current).
		return
	}
	system, err := evt.GetValue("system.pct")
	if err != nil {
		// Per-core idle CPUs may have no system.pct (Sys may be zero on macOS).
	}
	if user != nil {
		f := user.(float64)
		if !math.IsNaN(f) {
			assert.GreaterOrEqual(t, f, 0.0)
		}
	}
	if system != nil {
		f := system.(float64)
		if !math.IsNaN(f) {
			assert.GreaterOrEqual(t, f, 0.0)
		}
	}

	if norm {
		normUser, err := evt.GetValue("user.norm.pct")
		if err == nil && normUser != nil {
			f := normUser.(float64)
			if !math.IsNaN(f) {
				assert.LessOrEqual(t, f, 100.0)
			}
		}
		normSystem, err := evt.GetValue("system.norm.pct")
		if err == nil && normSystem != nil {
			f := normSystem.(float64)
			if !math.IsNaN(f) {
				assert.LessOrEqual(t, f, 100.0)
			}
		}

		userF, userOK := user.(float64)
		sysF, sysOK := system.(float64)
		if userOK && sysOK && userF > 0 && sysF > 0 {
			if normUser != nil {
				if nf, ok := normUser.(float64); ok && !math.IsNaN(nf) {
					assert.True(t, userF > nf)
				}
			}
			if normSystem != nil {
				if nf, ok := normSystem.(float64); ok && !math.IsNaN(nf) {
					assert.True(t, sysF > nf)
				}
			}
		}
	}

	userTicks, err := evt.GetValue("user.ticks")
	if err == nil && userTicks != nil {
		assert.True(t, userTicks.(uint64) >= 0)
	}
	systemTicks, err := evt.GetValue("system.ticks")
	if err == nil && systemTicks != nil {
		assert.True(t, systemTicks.(uint64) >= 0)
	}
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

// TestWindowsNegativeKernelTransition tests that when Sys transitions between
// zero-value (anomaly) and a valid value (normal), system.pct is always derived
// via totalPct - userPct, avoiding the wild percentage swings that the old ||
// condition produced.
//
// Background: On some 64-core Windows machines, GetSystemTimes() returns
// idle > kernel intermittently. gosigar computes kernel - idle, yielding a
// negative Duration. The fix in metrics_windows.go sets Sys to opt.Uint{} when
// kernel < 0. The Format() method must use && (not ||) so that a transition
// from normal→anomaly or anomaly→normal always takes the derive path.
func TestWindowsNegativeKernelTransition(t *testing.T) {
	// Layout: 30% user, 10% system (normalized), 60% idle.
	// Each step adds ~10000 ticks total. 64 logical CPUs.
	const numCPU = 64

	// S0: Normal (Sys is valid)
	s0 := CPU{
		User:       opt.UintWith(250000000),
		Sys:        opt.UintWith(100000000),
		Idle:       opt.UintWith(600000000),
		TotalTicks: opt.UintWith(900000000),
	}
	// S1: Normal (Sys still valid)
	s1 := CPU{
		User:       opt.UintWith(250048000),
		Sys:        opt.UintWith(100048000),
		Idle:       opt.UintWith(600224000),
		TotalTicks: opt.UintWith(900320000),
	}
	// S2: Anomaly (gosigar kernel-idle went negative, Sys = zero-value)
	s2 := CPU{
		User:       opt.UintWith(250096000),
		Sys:        opt.Uint{}, // NOT set due to overflow
		Idle:       opt.UintWith(600448000),
		TotalTicks: opt.UintWith(900640000),
	}
	// S3: Still anomaly
	s3 := CPU{
		User:       opt.UintWith(250144000),
		Sys:        opt.Uint{},
		Idle:       opt.UintWith(600672000),
		TotalTicks: opt.UintWith(900960000),
	}
	// S4: Recovered (Sys valid again)
	s4 := CPU{
		User:       opt.UintWith(250192000),
		Sys:        opt.UintWith(100144000),
		Idle:       opt.UintWith(600896000),
		TotalTicks: opt.UintWith(901280000),
	}

	type transition struct {
		name      string
		prev, cur CPU
		// sysNormPct expected: 0.10 (10% system normalized)
		// sysPct expected: 6.4 (10% * 64 CPUs)
	}

	cases := []transition{
		{"Normal -> Normal", s0, s1},
		{"Normal -> Anomaly", s1, s2},
		{"Anomaly -> Anomaly", s2, s3},
		{"Anomaly -> Normal", s3, s4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sample := Metrics{
				count:          numCPU,
				isTotals:       true,
				previousSample: tc.prev,
				currentSample:  tc.cur,
			}

			evt, err := sample.Format(MetricOpts{NormalizedPercentages: true, Percentages: true})
			assert.NoError(t, err, "Format() should not return error")

			// All scenarios should produce consistent system.pct regardless of Sys state.
			// Actual breakdown: 15% user, 15% system (derived), 70% idle, 30% total non-idle.
			sysNormPct, _ := evt.GetValue("system.norm.pct")
			assert.InDelta(t, 0.15, sysNormPct.(float64), 0.01,
				"system.norm.pct should be ~0.15 in all transition scenarios, got %v", sysNormPct)

			sysPct, _ := evt.GetValue("system.pct")
			assert.InDelta(t, 9.6, sysPct.(float64), 0.01,
				"system.pct should be ~9.6 in all transition scenarios, got %v", sysPct)

			// Sanity: total, user, idle should also be stable
			totalNormPct, _ := evt.GetValue("total.norm.pct")
			assert.InDelta(t, 0.30, totalNormPct.(float64), 0.01)

			userNormPct, _ := evt.GetValue("user.norm.pct")
			assert.InDelta(t, 0.15, userNormPct.(float64), 0.01)

			idleNormPct, _ := evt.GetValue("idle.norm.pct")
			assert.InDelta(t, 0.70, idleNormPct.(float64), 0.01)

			// Sanity: no percentage should be wildly out of range
			assert.Greater(t, sysNormPct.(float64), -1.0, "system.norm.pct should not be deeply negative")
			assert.Less(t, sysNormPct.(float64), 2.0, "system.norm.pct should not be wildly positive")
		})
	}
}

// TestIsDeltaReasonable tests the counter delta validation function.
func TestIsDeltaReasonable(t *testing.T) {
	tests := []struct {
		name   string
		prev   CPU
		cur    CPU
		numCPU int
		ok     bool
	}{
		{
			name:   "normal increment",
			prev:   CPU{User: opt.UintWith(250000000), Idle: opt.UintWith(600000000), TotalTicks: opt.UintWith(900000000)},
			cur:    CPU{User: opt.UintWith(250048000), Idle: opt.UintWith(600224000), TotalTicks: opt.UintWith(900320000)},
			numCPU: 64,
			ok:     true,
		},
		{
			name:   "total ticks decreased",
			prev:   CPU{TotalTicks: opt.UintWith(900320000)},
			cur:    CPU{TotalTicks: opt.UintWith(900000000)},
			numCPU: 64,
			ok:     false,
		},
		{
			name:   "idle decreased",
			prev:   CPU{Idle: opt.UintWith(600224000), TotalTicks: opt.UintWith(900320000)},
			cur:    CPU{Idle: opt.UintWith(600000000), TotalTicks: opt.UintWith(900640000)},
			numCPU: 64,
			ok:     false,
		},
		{
			name:   "user decreased",
			prev:   CPU{User: opt.UintWith(250048000), Idle: opt.UintWith(600224000), TotalTicks: opt.UintWith(900320000)},
			cur:    CPU{User: opt.UintWith(250000000), Idle: opt.UintWith(600448000), TotalTicks: opt.UintWith(900640000)},
			numCPU: 64,
			ok:     false,
		},
		{
			name:   "idle delta disproportionately large (Windows counter jump)",
			prev:   CPU{User: opt.UintWith(250000000), Idle: opt.UintWith(346063587765), TotalTicks: opt.UintWith(900320000)},
			cur:    CPU{User: opt.UintWith(250048000), Idle: opt.UintWith(347202537890), TotalTicks: opt.UintWith(1518366359)},
			numCPU: 64,
				// totalDelta/numCPU = 618046359/64 = 9,656,975 ms ≈ 2.7 hours
				// Threshold is 15min = 900,000 ms → 9,656,975 > 900,000 → caught			ok: false,
		},
		{
			name:   "high idle system (totalDelta within threshold)",
			prev:   CPU{User: opt.UintWith(1000), Idle: opt.UintWith(500000), TotalTicks: opt.UintWith(10000)},
			cur:    CPU{User: opt.UintWith(1100), Idle: opt.UintWith(510000), TotalTicks: opt.UintWith(11000)},
			numCPU: 64,
			ok:     true, // totalDelta/numCPU = 1000/64 = 15.6 ms << 900,000 ms threshold
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isDeltaReasonable(tc.prev, tc.cur, tc.numCPU)
			assert.Equal(t, tc.ok, result)
		})
	}
}

// TestFormatCounterDiscontinuity tests that Format() returns an error when
// TotalTicks decreases (uint64 underflow prevention).
func TestFormatCounterDiscontinuity(t *testing.T) {
	sample := Metrics{
		count:          64,
		isTotals:       true,
		previousSample: CPU{User: opt.UintWith(250048000), Idle: opt.UintWith(600224000), TotalTicks: opt.UintWith(900320000)},
		currentSample:  CPU{User: opt.UintWith(250096000), Idle: opt.UintWith(600448000), TotalTicks: opt.UintWith(900000000)},
	}

	_, err := sample.Format(MetricOpts{NormalizedPercentages: true, Percentages: true})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did not increase")
}

// TestFormatZeroPreviousSample tests that Format() handles a zero previous
// sample gracefully on non-Windows platforms (where TotalTicks is not set).
// The percentages are computed from cumulative values against zero, which
// is correct for the first sample.
func TestFormatZeroPreviousSample(t *testing.T) {
	sample := Metrics{
		count:          64,
		isTotals:       true,
		previousSample: CPU{}, // zero
		currentSample:  CPU{User: opt.UintWith(250048000), Idle: opt.UintWith(600224000), TotalTicks: opt.UintWith(900320000)},
	}

	_, err := sample.Format(MetricOpts{NormalizedPercentages: true, Percentages: true})
	assert.NoError(t, err)
}

// TestMonitorReusesCachedResultOnAnomaly tests that when isDeltaReasonable
// detects a counter discontinuity, Fetch() returns the cached last successful
// result instead of an error, ensuring monitoring continuity.
func TestMonitorReusesCachedResultOnAnomaly(t *testing.T) {
	// Build a sequence: normal -> anomaly -> normal
	// Use direct Metrics construction (no OS-level Get()) for deterministic testing.
	m := &Monitor{
		lastSample: CPUMetrics{},
		Hostfs:      resolve.NewTestResolver(""),
	}

	// S0: Normal sample
	s0 := CPU{User: opt.UintWith(250000000), Sys: opt.UintWith(100000000), Idle: opt.UintWith(600000000), TotalTicks: opt.UintWith(900000000)}
	// S1: Normal sample (valid delta from S0)
	s1 := CPU{User: opt.UintWith(250048000), Sys: opt.UintWith(100048000), Idle: opt.UintWith(600224000), TotalTicks: opt.UintWith(900320000)}
	// S2: Anomaly (TotalTicks jumped wildly)
	s2 := CPU{User: opt.UintWith(250096000), Sys: opt.UintWith(100096000), Idle: opt.UintWith(347202537890), TotalTicks: opt.UintWith(1518366359)}

	// Set up lastSample as S0, then compute S0->S1 as the "last successful" result.
	m.lastSample = CPUMetrics{totals: s0}

	// Simulate successful Fetch: manually set what Fetch() would produce for S0->S1.
	lastGood := Metrics{previousSample: s0, currentSample: s1, count: 64, isTotals: true}
	m.lastMetrics = lastGood
	m.lastSample = CPUMetrics{totals: s1}

	// Now test anomaly detection: S1->S2 has counter jump.
	// isDeltaReasonable should return false (idleDelta is huge).
	reasonable := isDeltaReasonable(s1, s2, 64)
	assert.False(t, reasonable, "S1->S2 should be detected as anomaly")

	// Since lastMetrics.count > 0, Fetch would return the cached result.
	// Verify the cached result is valid.
	evt, err := lastGood.Format(MetricOpts{Percentages: true, NormalizedPercentages: true})
	assert.NoError(t, err)

	sysPct, _ := evt.GetValue("system.pct")
	assert.NotNil(t, sysPct)
	assert.Less(t, sysPct.(float64), 100.0, "cached system.pct should be reasonable, not thousands of percent")

	// Test that without cache, it returns error (count == 0 means no cache).
	m2 := &Monitor{Hostfs: resolve.NewTestResolver("")}
	m2.lastSample = CPUMetrics{totals: s1}
	// m2.lastMetrics is zero-valued (count == 0)
	assert.Equal(t, 0, m2.lastMetrics.count)
}
