package proc

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type MemInfo struct {
	MemTotal     float64
	MemFree      float64
	MemAvailable float64
	SwapTotal    float64
	SwapFree     float64
	Buffers      float64
	Cached       float64
	SReclaimable float64
	Shmem        float64
}

func (m *MemInfo) Used() float64 {
	return m.MemTotal - m.MemFree - m.Buffers - m.Cached - m.SReclaimable
}

func (m *MemInfo) SwapUsed() float64 {
	return m.SwapTotal - m.SwapFree
}

var memInfoFd *os.File

func GetMemInfo() (*MemInfo, error) {
	if memInfoFd == nil {
		f, err := os.Open("/proc/meminfo")
		if err != nil {
			return nil, err
		}
		memInfoFd = f
	}
	if _, err := memInfoFd.Seek(0, 0); err != nil {
		memInfoFd.Close()
		memInfoFd = nil
		return nil, err
	}

	mi := &MemInfo{}
	scanner := bufio.NewScanner(memInfoFd)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		valStr = strings.TrimSuffix(valStr, " kB")
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			continue
		}
		switch name {
		case "MemTotal":
			mi.MemTotal = val
		case "MemFree":
			mi.MemFree = val
		case "MemAvailable":
			mi.MemAvailable = val
		case "SwapTotal":
			mi.SwapTotal = val
		case "SwapFree":
			mi.SwapFree = val
		case "Buffers":
			mi.Buffers = val
		case "Cached":
			mi.Cached = val
		case "SReclaimable":
			mi.SReclaimable = val
		case "Shmem":
			mi.Shmem = val
		}
	}
	return mi, scanner.Err()
}

func GetMemAvailable() (float64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemAvailable:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, err := strconv.ParseFloat(parts[1], 64)
				if err != nil {
					return 0, fmt.Errorf("parse MemAvailable: %w", err)
				}
				return val, nil
			}
		}
	}
	return 0, fmt.Errorf("MemAvailable not found")
}

func GetSwapInfo() (total, free float64, err error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		switch parts[0] {
		case "SwapTotal:":
			total, _ = strconv.ParseFloat(parts[1], 64)
		case "SwapFree:":
			free, _ = strconv.ParseFloat(parts[1], 64)
		}
	}
	return total, free, nil
}

func GetUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return 0, fmt.Errorf("empty uptime")
	}
	return strconv.ParseFloat(parts[0], 64)
}

func KBToMiB(kb float64) float64 {
	return kb / 1024
}

func Percent(ratio float64) float64 {
	return ratio * 100
}
