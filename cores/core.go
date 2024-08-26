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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
	"github.com/prometheus/procfs"
)

func NewCoreService(cfg *config.CGRConfig, caps *engine.Caps, fileCPU *os.File, stopChan chan struct{},
	shdWg *sync.WaitGroup, shdChan *utils.SyncedChan, connMgr *engine.ConnManager) *CoreService {
	var st *engine.CapsStats
	if caps.IsLimited() && cfg.CoreSCfg().CapsStatsInterval != 0 {
		st = engine.NewCapsStats(cfg.CoreSCfg().CapsStatsInterval, caps, stopChan)
	}
	cS := &CoreService{
		shdWg:     shdWg,
		shdChan:   shdChan,
		cfg:       cfg,
		CapsStats: st,
		fileCPU:   fileCPU,
		caps:      caps,
		connMgr:   connMgr,
	}
	go cS.computeAndSendMetrics(stopChan)
	return cS
}

type CoreService struct {
	cfg       *config.CGRConfig
	CapsStats *engine.CapsStats
	shdWg     *sync.WaitGroup
	shdChan   *utils.SyncedChan

	memProfMux   sync.Mutex
	finalMemProf string        // full path of the final memory profile created on stop/shutdown
	stopMemProf  chan struct{} // signal end of memory profiling

	fileCPUMux sync.Mutex
	fileCPU    *os.File

	caps    *engine.Caps
	connMgr *engine.ConnManager
}

// computeAndSendMetrics computes internal metrics at the specified interval and sends them to StatS.
// Continues to run until a signal is received on the stopChan channel.
func (cS *CoreService) computeAndSendMetrics(stopChan <-chan struct{}) {
	interval := cS.cfg.CoreSCfg().InternalMetricsInterval
	if interval == 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range 1 {
		select {
		case <-ticker.C:
			metrics := InternalMetrics{}
			_ = cS.V1Status(context.TODO(), nil, &metrics)
			b, err := json.Marshal(metrics)
			if err != nil {
				utils.Logger.Warning(fmt.Sprintf("<%s> %v", utils.CoreS, err))
			}
			var eventMetrics map[string]any
			if err := json.Unmarshal(b, &eventMetrics); err != nil {
				utils.Logger.Warning(fmt.Sprintf("<%s> %v", utils.CoreS, err))
			}
			cgrEv := &utils.CGREvent{
				Tenant: cS.cfg.GeneralCfg().DefaultTenant,
				ID:     utils.GenUUID(),
				Event:  eventMetrics,
				APIOpts: map[string]any{
					utils.MetaSubsys: utils.MetaCore,
				},
			}
			var reply []string // statqueue IDs
			if err := cS.connMgr.Call(context.TODO(), cS.cfg.CoreSCfg().StatSConns,
				utils.StatSv1ProcessEvent, cgrEv, &reply); err != nil {
				utils.Logger.Warning(fmt.Sprintf("<%s> failed to send internal metrics to StatS: %v", utils.CoreS, err))
			}
		case <-stopChan:
			return
		}
	}
}

// Shutdown is called to shutdown the service
func (cS *CoreService) Shutdown() {
	utils.Logger.Info(fmt.Sprintf("<%s> shutdown initialized", utils.CoreS))
	cS.StopMemoryProfiling() // safe to ignore error (irrelevant)
	utils.Logger.Info(fmt.Sprintf("<%s> shutdown complete", utils.CoreS))
}

// StartCPUProfiling starts CPU profiling and saves the profile to the specified path.
func (cS *CoreService) StartCPUProfiling(path string) error {
	if path == utils.EmptyString {
		return utils.NewErrMandatoryIeMissing("DirPath")
	}
	cS.fileCPUMux.Lock()
	defer cS.fileCPUMux.Unlock()

	if cS.fileCPU != nil {
		// Check if the profiling is already active by calling Stat() on the file handle.
		// If Stat() returns nil, it means profiling is already active.
		if _, err := cS.fileCPU.Stat(); err == nil {
			return errors.New("start CPU profiling: already started")
		}
	}
	file, err := StartCPUProfiling(path)
	if err != nil {
		return err
	}
	cS.fileCPU = file
	return nil
}

// StartCPUProfiling creates a file and passes it to pprof.StartCPUProfile. It returns the file
// to be able to verify the status of profiling and close it after profiling is stopped.
func StartCPUProfiling(path string) (*os.File, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("could not create CPU profile: %v", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		if err := f.Close(); err != nil {
			utils.Logger.Warning(fmt.Sprintf("<%s> %v", utils.CoreS, err))
		}
		return nil, fmt.Errorf("could not start CPU profile: %v", err)
	}
	return f, nil
}

// StopCPUProfiling stops CPU profiling and closes the profile file.
func (cS *CoreService) StopCPUProfiling() error {
	cS.fileCPUMux.Lock()
	defer cS.fileCPUMux.Unlock()
	pprof.StopCPUProfile()
	if cS.fileCPU == nil {
		return errors.New("stop CPU profiling: not started yet")
	}
	if err := cS.fileCPU.Close(); err != nil {
		if errors.Is(err, os.ErrClosed) {
			return errors.New("stop CPU profiling: already stopped")
		}
		return fmt.Errorf("could not close profile file: %v", err)
	}
	return nil
}

// MemoryProfilingParams represents the parameters for memory profiling.
type MemoryProfilingParams struct {
	Tenant   string
	DirPath  string        // directory path where memory profiles will be saved
	Interval time.Duration // duration between consecutive memory profile captures
	MaxFiles int           // maximum number of profile files to retain

	// UseTimestamp determines if the filename includes a timestamp.
	// The format is 'mem_20060102150405[_<microseconds>].prof'.
	// Microseconds are included if the interval is less than one second to avoid duplicate names.
	// If false, filenames follow an incremental format: 'mem_<n>.prof'.
	UseTimestamp bool

	APIOpts map[string]any
}

// StartMemoryProfiling starts memory profiling in the specified directory.
func (cS *CoreService) StartMemoryProfiling(params MemoryProfilingParams) error {
	if params.Interval <= 0 {
		params.Interval = 15 * time.Second
	}
	if params.MaxFiles < 0 {
		// consider any negative number to mean unlimited files
		params.MaxFiles = 0
	}

	cS.memProfMux.Lock()
	defer cS.memProfMux.Unlock()

	// Check if profiling is already started.
	select {
	case <-cS.stopMemProf: // triggered only on channel closed
	default:
		if cS.stopMemProf != nil {
			// stopMemProf being not closed and different from nil means that the profiling loop is already active.
			return errors.New("start memory profiling: already started")
		}
	}

	utils.Logger.Info(fmt.Sprintf(
		"<%s> starting memory profiling loop, writing to directory %q", utils.CoreS, params.DirPath))
	cS.stopMemProf = make(chan struct{})
	cS.finalMemProf = filepath.Join(params.DirPath, utils.MemProfFinalFile)
	cS.shdWg.Add(1)
	go cS.profileMemory(params)
	return nil
}

// newMemProfNameFunc returns a closure that generates memory profile filenames.
func newMemProfNameFunc(interval time.Duration, useTimestamp bool) func() string {
	if !useTimestamp {
		i := 0
		return func() string {
			i++
			return fmt.Sprintf("mem_%d.prof", i)
		}
	}
	if interval < time.Second {
		return func() string {
			now := time.Now()
			return fmt.Sprintf("mem_%s_%d.prof", now.Format("20060102150405"), now.Nanosecond()/1e3)
		}
	}

	return func() string {
		return fmt.Sprintf("mem_%s.prof", time.Now().Format("20060102150405"))
	}
}

// profileMemory runs the memory profiling loop, writing profiles to files at the specified interval.
func (cS *CoreService) profileMemory(params MemoryProfilingParams) {
	defer cS.shdWg.Done()
	fileName := newMemProfNameFunc(params.Interval, params.UseTimestamp)
	ticker := time.NewTicker(params.Interval)
	defer ticker.Stop()
	files := make([]string, 0, params.MaxFiles)
	for {
		select {
		case <-ticker.C:
			path := filepath.Join(params.DirPath, fileName())
			if err := writeHeapProfile(path); err != nil {
				utils.Logger.Err(fmt.Sprintf("<%s> %v", utils.CoreS, err))
				cS.StopMemoryProfiling()
			}
			if params.MaxFiles == 0 {
				// no file limit
				continue
			}
			if len(files) == params.MaxFiles {
				oldest := files[0]
				utils.Logger.Info(fmt.Sprintf("<%s> removing old heap profile file %q", utils.CoreS, oldest))
				files = files[1:] // remove oldest file from the list
				if err := os.Remove(oldest); err != nil {
					utils.Logger.Warning(fmt.Sprintf("<%s> %v", utils.CoreS, err))
				}
			}
			files = append(files, path)
		case <-cS.stopMemProf:
			if err := writeHeapProfile(cS.finalMemProf); err != nil {
				utils.Logger.Err(fmt.Sprintf("<%s> %v", utils.CoreS, err))
			}
			return
		}
	}
}

// writeHeapProfile writes the heap profile to the specified path.
func writeHeapProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not create memory profile: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			utils.Logger.Warning(fmt.Sprintf(
				"<%s> could not close file %q: %v", utils.CoreS, f.Name(), err))
		}
	}()
	utils.Logger.Info(fmt.Sprintf("<%s> writing heap profile to %q", utils.CoreS, path))
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("could not write memory profile: %v", err)
	}
	return nil
}

// StopMemoryProfiling stops memory profiling.
func (cS *CoreService) StopMemoryProfiling() error {
	cS.memProfMux.Lock()
	defer cS.memProfMux.Unlock()

	// Check if profiling is already stopped to prevent a channel close panic.
	select {
	case <-cS.stopMemProf: // triggered only on channel closed
		return errors.New("stop memory profiling: already stopped")
	default: // prevents blocking
		if cS.stopMemProf == nil {
			// stopMemProf being nil means that StartMemoryProfiling has never been called. There is nothing to stop.
			return errors.New("stop memory profiling: not started yet")
		}
	}

	utils.Logger.Info(fmt.Sprintf("<%s> stopping memory profiling loop", utils.CoreS))
	close(cS.stopMemProf)
	return nil
}

// V1Status returns the status of the engine
func (cS *CoreService) V1Status(_ *context.Context, _ *utils.TenantWithAPIOpts, reply *InternalMetrics) (err error) {
	metrics, err := computeInternalMetrics()
	if err != nil {
		return err
	}
	metrics.NodeID = cS.cfg.GeneralCfg().NodeID
	if cS.cfg.CoreSCfg().Caps != 0 {
		metrics.CapsStats.Allocated = cS.caps.Allocated()
		if cS.cfg.CoreSCfg().CapsStatsInterval != 0 {
			metrics.CapsStats.Peak = cS.CapsStats.GetPeak()
		}
	}
	*reply = metrics
	return
}

type InternalMetrics struct {
	GoVersion         string //`json:"go_info"`         // Gauge (make the value 1 and add the version as label)
	NodeID            string //`json:"cgrates_node_id"` // Gauge (make the value 1 and add the id as label)
	Version           string //`json:"cgrates_version"` // Gauge (make the value 1 and add the id as label)
	RunningSince      string
	Goroutines        int //`json:"go_goroutines"` // Gauge
	Threads           int //`json:"go_threads"`    // Gauge
	MemStats          GoMemStats
	GCDurationSeconds Summary //`json:"go_gc_duration_seconds"` // Summary
	ProcStats         ProcStats
	CapsStats         CapsStats
}

type GoMemStats struct {
	Alloc        uint64  //`json:"go_memstats_alloc_bytes"`          // Gauge
	TotalAlloc   uint64  //`json:"go_memstats_alloc_bytes_total"`    // Counter
	Sys          uint64  //`json:"go_memstats_sys_bytes"`            // Gauge
	Mallocs      uint64  //`json:"go_memstats_mallocs_total"`        // Counter
	Frees        uint64  //`json:"go_memstats_frees_total"`          // Counter
	Lookups      uint64  //`json:"go_memstats_lookups_total"`        // Counter
	HeapAlloc    uint64  //`json:"go_memstats_heap_alloc_bytes"`     // Gauge
	HeapSys      uint64  //`json:"go_memstats_heap_sys_bytes"`       // Gauge
	HeapIdle     uint64  //`json:"go_memstats_heap_idle_bytes"`      // Gauge
	HeapInuse    uint64  //`json:"go_memstats_heap_inuse_bytes"`     // Gauge
	HeapReleased uint64  //`json:"go_memstats_heap_released_bytes"`  // Gauge
	HeapObjects  uint64  //`json:"go_memstats_heap_objects"`         // Gauge
	StackInuse   uint64  //`json:"go_memstats_stack_inuse_bytes"`    // Gauge
	StackSys     uint64  //`json:"go_memstats_stack_sys_bytes"`      // Gauge
	MSpanSys     uint64  //`json:"go_memstats_mspan_sys_bytes"`      // Gauge
	MSpanInuse   uint64  //`json:"go_memstats_mspan_inuse_bytes"`    // Gauge
	MCacheInuse  uint64  //`json:"go_memstats_mcache_inuse_bytes"`   // Gauge
	MCacheSys    uint64  //`json:"go_memstats_mcache_sys_bytes"`     // Gauge
	BuckHashSys  uint64  //`json:"go_memstats_buck_hash_sys_bytes"`  // Gauge
	GCSys        uint64  //`json:"go_memstats_gc_sys_bytes"`         // Gauge
	OtherSys     uint64  //`json:"go_memstats_other_sys_bytes"`      // Gauge
	NextGC       uint64  //`json:"go_memstats_next_gc_bytes"`        // Gauge
	LastGC       float64 //`json:"go_memstats_last_gc_time_seconds"` // Gauge
}
type Summary struct { // Summary
	Quantiles []Quantile //`json:"go_gc_duration_seconds"`
	Sum       float64    //`json:"go_gc_duration_seconds_sum"`
	Count     uint64     //`json:"go_gc_duration_seconds_count"`
}
type Quantile struct {
	Quantile float64 //`json:"quantile"`
	Value    float64 //`json:"value"`
}

type ProcStats struct {
	CPUTime              float64 //`json:"process_cpu_seconds_total"`            // Counter
	MaxFDs               uint64  //`json:"process_max_fds"`                      // Gauge
	OpenFDs              int     //`json:"process_open_fds"`                     // Gauge
	ResidentMemory       int     //`json:"process_resident_memory_bytes"`        // Gauge
	StartTime            float64 //`json:"process_start_time_seconds"`           // Gauge
	VirtualMemory        uint    //`json:"process_virtual_memory_bytes"`         // Gauge
	MaxVirtualMemory     uint64  //`json:"process_virtual_memory_max_bytes"`     // Gauge
	NetworkReceiveTotal  float64 //`json:"process_network_receive_bytes_total"`  // Counter
	NetworkTransmitTotal float64 //`json:"process_network_transmit_bytes_total"` // Counter
}

type CapsStats struct {
	Allocated int //`json:"cgrates_caps_allocated"`
	Peak      int //`json:"cgrates_caps_peak"`
}

func computeInternalMetrics() (InternalMetrics, error) {
	vers, err := utils.GetCGRVersion()
	if err != nil {
		return InternalMetrics{}, err
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memStats := GoMemStats{
		Alloc:        m.Alloc,
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		Mallocs:      m.Mallocs,
		Frees:        m.Frees,
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		HeapIdle:     m.HeapIdle,
		HeapInuse:    m.HeapInuse,
		HeapReleased: m.HeapReleased,
		HeapObjects:  m.HeapObjects,
		StackInuse:   m.StackInuse,
		StackSys:     m.StackSys,
		MSpanInuse:   m.MSpanInuse,
		MSpanSys:     m.MSpanSys,
		MCacheInuse:  m.MCacheInuse,
		MCacheSys:    m.MCacheSys,
		BuckHashSys:  m.BuckHashSys,
		GCSys:        m.GCSys,
		OtherSys:     m.OtherSys,
		NextGC:       m.NextGC,
		Lookups:      m.Lookups,
	}

	threads, _ := runtime.ThreadCreateProfile(nil)

	var stats debug.GCStats
	stats.PauseQuantiles = make([]time.Duration, 5)
	debug.ReadGCStats(&stats)
	quantiles := make([]Quantile, 0, 5)
	// Add the first quantile separately
	quantiles = append(quantiles, Quantile{
		Quantile: 0.0,
		Value:    stats.PauseQuantiles[0].Seconds(),
	})
	for idx, pq := range stats.PauseQuantiles[1:] {
		q := Quantile{
			Quantile: float64(idx+1) / float64(len(stats.PauseQuantiles)-1),
			Value:    pq.Seconds(),
		}
		quantiles = append(quantiles, q)
	}
	gcDurSeconds := Summary{
		Quantiles: quantiles,
		Count:     uint64(stats.NumGC),
		Sum:       stats.PauseTotal.Seconds(),
	}
	memStats.LastGC = float64(stats.LastGC.UnixNano()) / 1e9

	// Process metrics
	pid := os.Getpid()
	p, err := procfs.NewProc(pid)
	if err != nil {
		return InternalMetrics{}, err
	}

	procStats := ProcStats{}
	if stat, err := p.Stat(); err == nil {
		procStats.CPUTime = stat.CPUTime()
		procStats.VirtualMemory = stat.VirtualMemory()
		procStats.ResidentMemory = stat.ResidentMemory()
		if startTime, err := stat.StartTime(); err == nil {
			procStats.StartTime = startTime
		} else {
			return InternalMetrics{}, err
		}
	} else {
		return InternalMetrics{}, err
	}
	if fds, err := p.FileDescriptorsLen(); err == nil {
		procStats.OpenFDs = fds
	} else {
		return InternalMetrics{}, err
	}

	if limits, err := p.Limits(); err == nil {
		procStats.MaxFDs = limits.OpenFiles
		procStats.MaxVirtualMemory = limits.AddressSpace
	} else {
		return InternalMetrics{}, err
	}

	if netstat, err := p.Netstat(); err == nil {
		var inOctets, outOctets float64
		if netstat.IpExt.InOctets != nil {
			inOctets = *netstat.IpExt.InOctets
		}
		if netstat.IpExt.OutOctets != nil {
			outOctets = *netstat.IpExt.OutOctets
		}
		procStats.NetworkReceiveTotal = inOctets
		procStats.NetworkTransmitTotal = outOctets
	} else {
		return InternalMetrics{}, err
	}

	return InternalMetrics{
		GoVersion:         runtime.Version(),
		Version:           vers,
		RunningSince:      utils.GetStartTime(),
		Goroutines:        runtime.NumGoroutine(),
		Threads:           threads,
		MemStats:          memStats,
		GCDurationSeconds: gcDurSeconds,
		ProcStats:         procStats,
	}, nil
}

// Sleep is used to test the concurrent requests mechanism
func (cS *CoreService) V1Sleep(_ *context.Context, arg *utils.DurationArgs, reply *string) error {
	time.Sleep(arg.Duration)
	*reply = utils.OK
	return nil
}

// StartCPUProfiling is used to start CPUProfiling in the given path
// V1StartCPUProfiling starts CPU profiling and saves the profile to the specified path.
func (cS *CoreService) V1StartCPUProfiling(_ *context.Context, args *utils.DirectoryArgs, reply *string) error {
	if err := cS.StartCPUProfiling(path.Join(args.DirPath, utils.CpuPathCgr)); err != nil {
		return err
	}
	*reply = utils.OK
	return nil
}

// StopCPUProfiling is used to stop CPUProfiling. The file should be written on the path
// where the CPUProfiling already started
func (cS *CoreService) V1StopCPUProfiling(_ *context.Context, _ *utils.TenantWithAPIOpts, reply *string) error {
	if err := cS.StopCPUProfiling(); err != nil {
		return err
	}
	*reply = utils.OK
	return nil
}

// V1StartMemoryProfiling starts memory profiling in the specified directory.
func (cS *CoreService) V1StartMemoryProfiling(_ *context.Context, params MemoryProfilingParams, reply *string) error {
	if params.DirPath == utils.EmptyString {
		return utils.NewErrMandatoryIeMissing("DirPath")
	}
	if err := cS.StartMemoryProfiling(params); err != nil {
		return err
	}
	*reply = utils.OK
	return nil
}

// V1StopMemoryProfiling stops memory profiling.
func (cS *CoreService) V1StopMemoryProfiling(_ *context.Context, _ utils.TenantWithAPIOpts, reply *string) error {
	if err := cS.StopMemoryProfiling(); err != nil {
		return err
	}
	*reply = utils.OK
	return nil
}

// V1Panic is used print the Message sent as a panic
func (cS *CoreService) V1Panic(_ *context.Context, args *utils.PanicMessageArgs, _ *string) error {
	panic(args.Message)
}
