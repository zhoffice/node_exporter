// Copyright 2016 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !noprocess

package collector

import (
	"github.com/percona/gopsutil/process"
	"github.com/prometheus/client_golang/prometheus"
)

var visibleStatNames = map[string][]string{
	"memory":     []string{"used"},
	"cpu":        []string{"total", "user", "system"},
	"virtualmem": []string{"peak", "size", "lock", "hwm", "rss", "swap"},
}

type procInfo map[string]float64

type processCollector struct{}

func init() {
	Factories["process"] = newProcessCollector
}

func newProcessCollector() (Collector, error) {
	return &processCollector{}, nil
}

// Update implements "process" collector scraping
func (p processCollector) Update(ch chan<- prometheus.Metric) error {
	procs := map[string]procInfo{}

	pids, err := process.Pids()
	if err != nil {
		return err
	}

	// Group pids by processes and collect usage info.
	for _, pid := range pids {
		proc, err := process.NewProcess(pid)
		if err != nil {
			continue
		}
		procName, err := proc.Name()
		if err != nil || procName == "" {
			continue
		}

		if _, ok := procs[procName]; !ok {
			procs[procName] = procInfo{}
		}

		if memInfo, err := proc.MemoryInfo(); err == nil {
			procs[procName]["memory.used"] += float64(memInfo.RSS)
		}

		if virtualMemInfo, err := proc.VirtualMemoryInfo(); err == nil {
			procs[procName]["virtualmem.peak"] += float64(virtualMemInfo.VMPeak)
			procs[procName]["virtualmem.size"] += float64(virtualMemInfo.VMSize)
			procs[procName]["virtualmem.lock"] += float64(virtualMemInfo.VMLck)
			procs[procName]["virtualmem.hwm"] += float64(virtualMemInfo.VMHWM)
			procs[procName]["virtualmem.rss"] += float64(virtualMemInfo.VMRSS)
			procs[procName]["virtualmem.swap"] += float64(virtualMemInfo.VMSwap)
		}

		stats, err := proc.GetStats()
		if err != nil {
			continue
		}
		if timesStat, err := proc.Times(); err == nil {
			procs[procName]["cpu.user"] += float64(stats.UserTime)
			procs[procName]["cpu.system"] += timesStat.System
			procs[procName]["cpu.total"] += timesStat.Total()
		}

	}

	// Build metrics.
	for procName, proc := range procs {
		for system, subsystems := range visibleStatNames {
			desc := prometheus.NewDesc(
				prometheus.BuildFQName(Namespace, "process", system),
				"Labeled per process cpu and memory information.",
				[]string{"name", "type"},
				nil,
			)
			for _, s := range subsystems {
				metricType := system + "." + s
				ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(proc[metricType]), procName, s)
			}
		}
	}

	return nil
}
