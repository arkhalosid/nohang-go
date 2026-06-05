package main

import (
	"flag"
	"fmt"
	"github.com/user/nohang/internal/proc"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	metrics := flag.String("m", "memory", "metrics (memory, io or cpu)")
	flag.Parse()

	met := *metrics
	if met != "memory" && met != "io" && met != "cpu" {
		fmt.Fprintf(os.Stderr, "ERROR: invalid metrics: %s\n", met)
		os.Exit(1)
	}

	_, err := proc.ReadPSIFile("/proc/pressure/memory")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: PSI not available: %v\n", err)
		os.Exit(1)
	}

	mountPoint := findCGroup2Mount()
	if mountPoint == "" {
		fmt.Fprintln(os.Stderr, "ERROR: cgroup2 not mounted")
		os.Exit(1)
	}

	psiPath := fmt.Sprintf("/proc/pressure/%s", met)

	if met == "cpu" {
		fmt.Printf("PSI metrics: %s\ncgroup_v2 mountpoint: %s\n", met, mountPoint)
		fmt.Println("=====================|")
		fmt.Println("         some        |")
		fmt.Println("-------------------- |")
		fmt.Println(" avg10  avg60 avg300 | cgroup_v2")
		fmt.Println("------ ------ ------ | -----------")

		m, err := proc.ReadPSIFile(psiPath)
		if err == nil {
			fmt.Printf("%6.2f %6.2f %6.2f | SYSTEM_WIDE\n", m.SomeAvg10, m.SomeAvg60, m.SomeAvg300)
		}

		err = filepath.Walk(mountPoint, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if strings.HasSuffix(path, "/cpu.pressure") {
				m, err := proc.ReadPSIFile(path)
				if err != nil {
					return nil
				}
				cg := strings.TrimPrefix(path, mountPoint)
				cg = strings.TrimSuffix(cg, "/cpu.pressure")
				fmt.Printf("%6.2f %6.2f %6.2f | %s\n", m.SomeAvg10, m.SomeAvg60, m.SomeAvg300, cg)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "walk error: %v\n", err)
		}
	} else {
		fmt.Printf("PSI metrics: %s\ncgroup_v2 mountpoint: %s\n", met, mountPoint)
		fmt.Println("=====================|======================|")
		fmt.Println("         some        |          full        |")
		fmt.Println("-------------------- | -------------------- |")
		fmt.Println(" avg10  avg60 avg300 |  avg10  avg60 avg300 | cgroup_v2")
		fmt.Println("------ ------ ------ | ------ ------ ------ | -----------")

		m, err := proc.ReadPSIFile(psiPath)
		if err == nil {
			fmt.Printf("%6.2f %6.2f %6.2f | %6.2f %6.2f %6.2f | SYSTEM_WIDE\n",
				m.SomeAvg10, m.SomeAvg60, m.SomeAvg300,
				m.FullAvg10, m.FullAvg60, m.FullAvg300)
		}

		ext := fmt.Sprintf("/%s.pressure", met)
		filepath.Walk(mountPoint, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if strings.HasSuffix(path, ext) {
				m, err := proc.ReadPSIFile(path)
				if err != nil {
					return nil
				}
				cg := strings.TrimPrefix(path, mountPoint)
				cg = strings.TrimSuffix(cg, ext)
				fmt.Printf("%6.2f %6.2f %6.2f | %6.2f %6.2f %6.2f | %s\n",
					m.SomeAvg10, m.SomeAvg60, m.SomeAvg300,
					m.FullAvg10, m.FullAvg60, m.FullAvg300, cg)
			}
			return nil
		})
	}
}

func findCGroup2Mount() string {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "cgroup2") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}
