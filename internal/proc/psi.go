package proc

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type PSIMetrics struct {
	SomeAvg10  float64
	SomeAvg60  float64
	SomeAvg300 float64
	FullAvg10  float64
	FullAvg60  float64
	FullAvg300 float64
}

var psiPaths = map[string]string{
	"cpu":    "/proc/pressure/cpu",
	"io":     "/proc/pressure/io",
	"memory": "/proc/pressure/memory",
}

func ReadPSIFile(path string) (*PSIMetrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := &PSIMetrics{}
	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid PSI format")
	}
	parseLine := func(line string) (avg10, avg60, avg300 float64, err error) {
		parts := strings.Fields(line)
		for _, p := range parts {
			if strings.HasPrefix(p, "avg10=") {
				avg10, _ = strconv.ParseFloat(p[6:], 64)
			} else if strings.HasPrefix(p, "avg60=") {
				avg60, _ = strconv.ParseFloat(p[6:], 64)
			} else if strings.HasPrefix(p, "avg300=") {
				avg300, _ = strconv.ParseFloat(p[7:], 64)
			}
		}
		return
	}
	m.SomeAvg10, m.SomeAvg60, m.SomeAvg300, _ = parseLine(lines[0])
	if len(lines) > 1 && lines[1] != "" {
		m.FullAvg10, m.FullAvg60, m.FullAvg300, _ = parseLine(lines[1])
	}
	return m, nil
}

func ReadPSIValue(path, metric string) (float64, error) {
	m, err := ReadPSIFile(path)
	if err != nil {
		return 0, err
	}
	switch metric {
	case "some_avg10":
		return m.SomeAvg10, nil
	case "some_avg60":
		return m.SomeAvg60, nil
	case "some_avg300":
		return m.SomeAvg300, nil
	case "full_avg10":
		return m.FullAvg10, nil
	case "full_avg60":
		return m.FullAvg60, nil
	case "full_avg300":
		return m.FullAvg300, nil
	}
	return 0, fmt.Errorf("unknown PSI metric: %s", metric)
}

func PSIKernelOK() bool {
	_, err := ReadPSIFile("/proc/pressure/memory")
	return err == nil
}
