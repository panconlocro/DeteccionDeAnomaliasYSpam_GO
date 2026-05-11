package benchmark

import (
	"runtime"
	"syscall"
	"time"
)

type ResourceUsage struct {
	WallSeconds       float64 `json:"wall_seconds"`
	CPUPercent        float64 `json:"cpu_percent"`
	CPUUserSeconds    float64 `json:"cpu_user_seconds"`
	CPUSystemSeconds  float64 `json:"cpu_system_seconds"`
	CPUTotalSeconds   float64 `json:"cpu_total_seconds"`
	MaxRSSMB          float64 `json:"max_rss_mb"`
	MaxHeapAllocMB    float64 `json:"max_heap_alloc_mb"`
	MaxHeapSysMB      float64 `json:"max_heap_sys_mb"`
	MaxGoSysMB        float64 `json:"max_go_sys_mb"`
	TotalAllocDeltaMB float64 `json:"total_alloc_delta_mb"`
	NumGCDelta        float64 `json:"num_gc_delta"`
}

type resourceSnapshot struct {
	CPUUserSeconds   float64
	CPUSystemSeconds float64
	MaxRSS           uint64
	HeapAlloc        uint64
	HeapSys          uint64
	GoSys            uint64
	TotalAlloc       uint64
	NumGC            uint32
}

func StartResourceMonitor(interval time.Duration) func() ResourceUsage {
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}

	start := time.Now()
	before := readResourceSnapshot()
	stop := make(chan struct{})
	done := make(chan ResourceUsage, 1)

	go func() {
		maxRSS := before.MaxRSS
		maxHeapAlloc := before.HeapAlloc
		maxHeapSys := before.HeapSys
		maxGoSys := before.GoSys

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		updateMax := func(snapshot resourceSnapshot) {
			if snapshot.MaxRSS > maxRSS {
				maxRSS = snapshot.MaxRSS
			}
			if snapshot.HeapAlloc > maxHeapAlloc {
				maxHeapAlloc = snapshot.HeapAlloc
			}
			if snapshot.HeapSys > maxHeapSys {
				maxHeapSys = snapshot.HeapSys
			}
			if snapshot.GoSys > maxGoSys {
				maxGoSys = snapshot.GoSys
			}
		}

		for {
			select {
			case <-ticker.C:
				updateMax(readResourceSnapshot())
			case <-stop:
				after := readResourceSnapshot()
				updateMax(after)
				wallSeconds := time.Since(start).Seconds()
				cpuTotalSeconds := after.CPUUserSeconds + after.CPUSystemSeconds - before.CPUUserSeconds - before.CPUSystemSeconds
				cpuPercent := 0.0
				if wallSeconds > 0 {
					cpuPercent = (cpuTotalSeconds / wallSeconds) * 100
				}

				done <- ResourceUsage{
					WallSeconds:       wallSeconds,
					CPUPercent:        cpuPercent,
					CPUUserSeconds:    after.CPUUserSeconds - before.CPUUserSeconds,
					CPUSystemSeconds:  after.CPUSystemSeconds - before.CPUSystemSeconds,
					CPUTotalSeconds:   cpuTotalSeconds,
					MaxRSSMB:          bytesToMB(maxRSS),
					MaxHeapAllocMB:    bytesToMB(maxHeapAlloc),
					MaxHeapSysMB:      bytesToMB(maxHeapSys),
					MaxGoSysMB:        bytesToMB(maxGoSys),
					TotalAllocDeltaMB: bytesToMB(after.TotalAlloc - before.TotalAlloc),
					NumGCDelta:        float64(after.NumGC - before.NumGC),
				}
				return
			}
		}
	}()

	return func() ResourceUsage {
		close(stop)
		return <-done
	}
}

func readResourceSnapshot() resourceSnapshot {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	userSeconds, systemSeconds, maxRSS := readRusage()

	return resourceSnapshot{
		CPUUserSeconds:   userSeconds,
		CPUSystemSeconds: systemSeconds,
		MaxRSS:           maxRSS,
		HeapAlloc:        mem.HeapAlloc,
		HeapSys:          mem.HeapSys,
		GoSys:            mem.Sys,
		TotalAlloc:       mem.TotalAlloc,
		NumGC:            mem.NumGC,
	}
}

func readRusage() (float64, float64, uint64) {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return 0, 0, 0
	}

	return timevalSeconds(usage.Utime), timevalSeconds(usage.Stime), maxRSSBytes(usage.Maxrss)
}

func timevalSeconds(value syscall.Timeval) float64 {
	return float64(value.Sec) + float64(value.Usec)/1_000_000
}

func bytesToMB(value uint64) float64 {
	return float64(value) / 1024 / 1024
}

func maxRSSBytes(value int64) uint64 {
	if value < 0 {
		return 0
	}

	// macOS reports ru_maxrss in bytes; Linux reports it in kilobytes.
	if runtime.GOOS == "darwin" {
		return uint64(value)
	}
	return uint64(value) * 1024
}
