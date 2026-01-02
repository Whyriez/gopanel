package handlers

import (
	"fmt"
	"math"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func GetSystemInfo(c *fiber.Ctx) error {
	v, _ := mem.VirtualMemory()
	c_percent, _ := cpu.Percent(1*time.Second, false)
	d, _ := disk.Usage("/")

	return c.JSON(fiber.Map{
		"cpu": fiber.Map{
			"usage_percent": math.Round(c_percent[0]*100) / 100,
		},
		"ram": fiber.Map{
			"total":        formatBytes(v.Total),
			"used":         formatBytes(v.Used),
			"used_percent": math.Round(v.UsedPercent*100) / 100,
		},
		"disk": fiber.Map{
			"total":        formatBytes(d.Total),
			"used":         formatBytes(d.Used),
			"used_percent": math.Round(d.UsedPercent*100) / 100,
		},
	})
}