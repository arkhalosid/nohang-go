package proc

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func CheckZRAM() float64 {
	if _, err := os.Stat("/sys/block/zram0/mem_limit"); os.IsNotExist(err) {
		return 0
	}
	devices, err := os.ReadDir("/sys/block")
	if err != nil {
		return 0
	}
	var sum float64
	for _, dev := range devices {
		if !strings.HasPrefix(dev.Name(), "zram") {
			continue
		}
		if data, err := os.ReadFile(fmt.Sprintf("/sys/block/%s/mm_stat", dev.Name())); err == nil {
			parts := strings.Fields(string(data))
			if len(parts) >= 3 {
				v, _ := strconv.ParseFloat(parts[2], 64)
				sum += v
			}
		} else if data, err := os.ReadFile(fmt.Sprintf("/sys/block/%s/mem_used_total", dev.Name())); err == nil {
			v, _ := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
			sum += v
		}
	}
	return sum / 1024
}
