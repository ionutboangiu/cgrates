/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package cores

import (
	"os"
	"reflect"
	"runtime"
	"testing"

	"github.com/cgrates/cgrates/utils"
	"github.com/prometheus/procfs"
)

func TestStatusMetricsToMap(t *testing.T) {
	memStats := GoMemStats{
		Alloc:        20,
		TotalAlloc:   100,
		Sys:          1,
		Mallocs:      1,
		Frees:        1,
		HeapAlloc:    1000,
		HeapSys:      10,
		HeapIdle:     500,
		HeapInuse:    10,
		HeapReleased: 300,
		HeapObjects:  10,
		StackInuse:   300,
		StackSys:     10,
		MSpanSys:     200,
		MSpanInuse:   1,
		MCacheInuse:  30,
		MCacheSys:    300,
		BuckHashSys:  20,
		GCSys:        30,
		OtherSys:     30,
		NextGC:       40,
		LastGC:       40.4,
	}
	gcDurationStats := GCDurationStats{}
	procStats := ProcStats{}
	capsStats := &CapsStats{}

	sm := StatusMetrics{
		PID:             1234,
		GoVersion:       "go1.16",
		NodeID:          "node123",
		Version:         "v1.0.0",
		Goroutines:      10,
		Threads:         5,
		MemStats:        memStats,
		GCDurationStats: gcDurationStats,
		ProcStats:       procStats,
		CapsStats:       capsStats,
		MaxProcs:        3,
		GCPercent:       100,
		MemLimit:        5555,
	}

	result, err := sm.toMap(true, "UTC")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := map[string]any{
		utils.PID:                     1234.,
		utils.GoVersion:               "go1.16",
		utils.NodeID:                  "node123",
		utils.FieldVersion:            "v1.0.0",
		utils.MetricRuntimeGoroutines: 10.,
		utils.MetricRuntimeThreads:    5.,
		utils.FieldMemStats:           memStats.toMap(),
		utils.FieldGCDurationStats:    gcDurationStats.toMap(),
		utils.FieldProcStats:          procStats.toMap(),
		utils.FieldCapsStats:          capsStats.toMap(),
		utils.MetricRuntimeMaxProcs:   3.,
		utils.MetricGCPercent:         100.,
		utils.MetricMemLimit:          5555.,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", utils.ToJSON(expected), utils.ToJSON(result))
	}

	condensedResult, err := sm.toMap(false, "UTC")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if condensedResult == nil {
		t.Errorf("Expected non-nil map for debug=false")
	}
}

func TestComputeAppMetrics(t *testing.T) {
	metrics, err := computeAppMetrics()
	if err != nil {
		t.Fatalf("computeAppMetrics returned an error: %v", err)
	}

	if metrics.PID != float64(os.Getpid()) {
		t.Errorf("Expected PID %d, but got %g", os.Getpid(), metrics.PID)
	}

	if metrics.GoVersion != runtime.Version() {
		t.Errorf("Expected GoVersion %s, but got %s", runtime.Version(), metrics.GoVersion)
	}

	p, err := procfs.NewProc(int(metrics.PID))
	if err != nil {
		t.Fatalf("Failed to create procfs proc: %v", err)
	}

	stat, err := p.Stat()
	if err != nil {
		t.Fatalf("Failed to get proc stat: %v", err)
	}

	if metrics.ProcStats.VirtualMemory != float64(stat.VirtualMemory()) {
		t.Errorf("Expected VirtualMemory %d, but got %g", stat.VirtualMemory(), metrics.ProcStats.VirtualMemory)
	}

}
