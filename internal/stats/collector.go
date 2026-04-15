// Package stats collects CPU, memory, disk and VRAM metrics and logs them
// periodically. It is activated only when workstation.enabled = true in
// config.toml so the same binary can run on both the Pi and the workstation.
package stats

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/digiogithub/smartworkstation/internal/config"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// Snapshot holds one point-in-time measurement.
type Snapshot struct {
	CPUPercent  float64
	MemUsed     uint64
	MemTotal    uint64
	DiskUsed    uint64
	DiskTotal   uint64
	VRAMUsed    uint64
	VRAMTotal   uint64
	VRAMSource  string
}

// String formats the snapshot for human-readable log output.
func (s *Snapshot) String() string {
	line := fmt.Sprintf(
		"CPU: %5.1f%%  |  RAM: %s / %s  |  Disk: %s / %s",
		s.CPUPercent,
		fmtBytes(s.MemUsed), fmtBytes(s.MemTotal),
		fmtBytes(s.DiskUsed), fmtBytes(s.DiskTotal),
	)
	if s.VRAMTotal > 0 {
		line += fmt.Sprintf("  |  VRAM(%s): %s / %s",
			s.VRAMSource, fmtBytes(s.VRAMUsed), fmtBytes(s.VRAMTotal))
	}
	return line
}

// Collector runs the periodic stats loop.
type Collector struct {
	cfg *config.Config
}

// NewCollector creates a Collector bound to cfg.
func NewCollector(cfg *config.Config) *Collector {
	return &Collector{cfg: cfg}
}

// Start collects metrics immediately and then on every StatsInterval tick until
// ctx is cancelled.
func (c *Collector) Start(ctx context.Context) {
	interval := time.Duration(c.cfg.Workstation.StatsInterval) * time.Second
	log.Printf("[stats] collector started — interval %v", interval)

	c.report()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[stats] collector stopped")
			return
		case <-ticker.C:
			c.report()
		}
	}
}

func (c *Collector) report() {
	snap, err := collect()
	if err != nil {
		log.Printf("[stats] collection error: %v", err)
		return
	}
	log.Printf("[stats] %s", snap)
}

// collect gathers all metrics and returns a Snapshot.
func collect() (*Snapshot, error) {
	s := &Snapshot{}

	// CPU — measure over 1 second for accuracy.
	pcts, err := cpu.Percent(time.Second, false)
	if err == nil && len(pcts) > 0 {
		s.CPUPercent = pcts[0]
	}

	// Memory
	if vm, err := mem.VirtualMemory(); err == nil {
		s.MemUsed = vm.Used
		s.MemTotal = vm.Total
	}

	// Disk (root filesystem)
	if du, err := disk.Usage("/"); err == nil {
		s.DiskUsed = du.Used
		s.DiskTotal = du.Total
	}

	// VRAM — implemented per-platform (vram_linux.go / vram_other.go)
	s.VRAMUsed, s.VRAMTotal, s.VRAMSource = detectVRAM()

	return s, nil
}

// fmtBytes converts a byte count to a human-readable string (KiB … GiB).
func fmtBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
