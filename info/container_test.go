// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package info

import (
	"testing"
	"time"
)

func TestStatsStartTime(t *testing.T) {
	N := 10
	stats := make([]*ContainerStats, 0, N)
	ct := time.Now()
	for i := 0; i < N; i++ {
		s := &ContainerStats{
			Timestamp: ct.Add(time.Duration(i) * time.Second),
		}
		stats = append(stats, s)
	}
	cinfo := &ContainerInfo{
		Name:  "/some/container",
		Stats: stats,
	}
	ref := ct.Add(time.Duration(N-1) * time.Second)
	end := cinfo.StatsEndTime()

	if !ref.Equal(end) {
		t.Errorf("end time is %v; should be %v", end, ref)
	}
}

func TestStatsEndTime(t *testing.T) {
	N := 10
	stats := make([]*ContainerStats, 0, N)
	ct := time.Now()
	for i := 0; i < N; i++ {
		s := &ContainerStats{
			Timestamp: ct.Add(time.Duration(i) * time.Second),
		}
		stats = append(stats, s)
	}
	cinfo := &ContainerInfo{
		Name:  "/some/container",
		Stats: stats,
	}
	ref := ct
	start := cinfo.StatsStartTime()

	if !ref.Equal(start) {
		t.Errorf("start time is %v; should be %v", start, ref)
	}
}

func TestPercentiles(t *testing.T) {
	N := 100
	data := make([]uint64, N)

	for i := 0; i < N; i++ {
		data[i] = uint64(i)
	}
	percentages := []int{
		80,
		90,
		50,
	}
	percentiles := uint64Slice(data).Percentiles(percentages...)
	for i, s := range percentiles {
		p := percentages[i]
		d := uint64(N * p / 100)
		if d != s {
			t.Errorf("%v percentile data should be %v, but got %v", p, d, s)
		}
	}
}

func TestNewSampleNilStats(t *testing.T) {
	stats := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	stats.Cpu.Usage.PerCpu = []uint64{uint64(10)}
	stats.Cpu.Usage.Total = uint64(10)
	stats.Cpu.Usage.System = uint64(2)
	stats.Cpu.Usage.User = uint64(8)
	stats.Memory.Usage = uint64(200)

	sample, err := NewSample(nil, stats)
	if err == nil {
		t.Errorf("generated an unexpected sample: %+v", sample)
	}

	sample, err = NewSample(stats, nil)
	if err == nil {
		t.Errorf("generated an unexpected sample: %+v", sample)
	}
}

func TestAddSample(t *testing.T) {
	cpuPrevUsage := uint64(10)
	cpuCurrentUsage := uint64(15)
	memCurrentUsage := uint64(200)

	prev := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	prev.Cpu.Usage.PerCpu = []uint64{cpuPrevUsage}
	prev.Cpu.Usage.Total = cpuPrevUsage
	prev.Cpu.Usage.System = 0
	prev.Cpu.Usage.User = cpuPrevUsage
	prev.Timestamp = time.Now()

	current := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	current.Cpu.Usage.PerCpu = []uint64{cpuCurrentUsage}
	current.Cpu.Usage.Total = cpuCurrentUsage
	current.Cpu.Usage.System = 0
	current.Cpu.Usage.User = cpuCurrentUsage
	current.Memory.Usage = memCurrentUsage
	current.Timestamp = prev.Timestamp.Add(1 * time.Second)

	sample, err := NewSample(prev, current)
	if err != nil {
		t.Errorf("should be able to generate a sample. but received error: %v", err)
	}
	if sample == nil {
		t.Fatalf("nil sample and nil error. unexpected result!")
	}

	if sample.Memory.Usage != memCurrentUsage {
		t.Errorf("wrong memory usage: %v. should be %v", sample.Memory.Usage, memCurrentUsage)
	}

	if sample.Cpu.Usage != cpuCurrentUsage-cpuPrevUsage {
		t.Errorf("wrong CPU usage: %v. should be %v", sample.Cpu.Usage, cpuCurrentUsage-cpuPrevUsage)
	}
}

func TestAddSampleIncompleteStats(t *testing.T) {
	cpuPrevUsage := uint64(10)
	cpuCurrentUsage := uint64(15)
	memCurrentUsage := uint64(200)

	prev := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	prev.Cpu.Usage.PerCpu = []uint64{cpuPrevUsage}
	prev.Cpu.Usage.Total = cpuPrevUsage
	prev.Cpu.Usage.System = 0
	prev.Cpu.Usage.User = cpuPrevUsage
	prev.Timestamp = time.Now()

	current := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	current.Cpu.Usage.PerCpu = []uint64{cpuCurrentUsage}
	current.Cpu.Usage.Total = cpuCurrentUsage
	current.Cpu.Usage.System = 0
	current.Cpu.Usage.User = cpuCurrentUsage
	current.Memory.Usage = memCurrentUsage
	current.Timestamp = prev.Timestamp.Add(1 * time.Second)

	stats := &ContainerStats{
		Cpu:    prev.Cpu,
		Memory: nil,
	}
	sample, err := NewSample(stats, current)
	if err == nil {
		t.Errorf("generated an unexpected sample: %+v", sample)
	}
	sample, err = NewSample(prev, stats)
	if err == nil {
		t.Errorf("generated an unexpected sample: %+v", sample)
	}

	stats = &ContainerStats{
		Cpu:    nil,
		Memory: prev.Memory,
	}
	sample, err = NewSample(stats, current)
	if err == nil {
		t.Errorf("generated an unexpected sample: %+v", sample)
	}
	sample, err = NewSample(prev, stats)
	if err == nil {
		t.Errorf("generated an unexpected sample: %+v", sample)
	}
}

func TestAddSampleWrongOrder(t *testing.T) {
	cpuPrevUsage := uint64(10)
	cpuCurrentUsage := uint64(15)
	memCurrentUsage := uint64(200)

	prev := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	prev.Cpu.Usage.PerCpu = []uint64{cpuPrevUsage}
	prev.Cpu.Usage.Total = cpuPrevUsage
	prev.Cpu.Usage.System = 0
	prev.Cpu.Usage.User = cpuPrevUsage
	prev.Timestamp = time.Now()

	current := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	current.Cpu.Usage.PerCpu = []uint64{cpuCurrentUsage}
	current.Cpu.Usage.Total = cpuCurrentUsage
	current.Cpu.Usage.System = 0
	current.Cpu.Usage.User = cpuCurrentUsage
	current.Memory.Usage = memCurrentUsage
	current.Timestamp = prev.Timestamp.Add(1 * time.Second)

	sample, err := NewSample(current, prev)
	if err == nil {
		t.Errorf("generated an unexpected sample: %+v", sample)
	}
}

func TestAddSampleWrongCpuUsage(t *testing.T) {
	cpuPrevUsage := uint64(15)
	cpuCurrentUsage := uint64(10)
	memCurrentUsage := uint64(200)

	prev := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	prev.Cpu.Usage.PerCpu = []uint64{cpuPrevUsage}
	prev.Cpu.Usage.Total = cpuPrevUsage
	prev.Cpu.Usage.System = 0
	prev.Cpu.Usage.User = cpuPrevUsage
	prev.Timestamp = time.Now()

	current := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	current.Cpu.Usage.PerCpu = []uint64{cpuCurrentUsage}
	current.Cpu.Usage.Total = cpuCurrentUsage
	current.Cpu.Usage.System = 0
	current.Cpu.Usage.User = cpuCurrentUsage
	current.Memory.Usage = memCurrentUsage
	current.Timestamp = prev.Timestamp.Add(1 * time.Second)

	sample, err := NewSample(prev, current)
	if err == nil {
		t.Errorf("generated an unexpected sample: %+v", sample)
	}
}
