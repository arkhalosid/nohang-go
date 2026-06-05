package main

import (
	"flag"
	"fmt"
	"github.com/user/nohang/internal/proc"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var peaks = make(map[string]float64)

func main() {
	target := flag.String("t", "SYSTEM_WIDE", "target (cgroup_v2 or SYSTEM_WIDE)")
	interval := flag.Float64("i", 2, "interval in sec")
	flag.String("l", "", "path to log file")
	mode := flag.String("m", "0", "mode (0, 1 or 2)")
	suppress := flag.String("s", "False", "suppress output")
	flag.Parse()

	if *interval < 1 {
		fmt.Fprintln(os.Stderr, "error: interval must be >= 1")
		os.Exit(1)
	}
	if *mode != "0" && *mode != "1" && *mode != "2" {
		fmt.Fprintln(os.Stderr, "ERROR: invalid mode")
		os.Exit(1)
	}

	_, err := proc.ReadPSIFile("/proc/pressure/memory")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: PSI not available: %v\n", err)
		os.Exit(1)
	}

	suppressOutput := *suppress == "True"

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP)
	go func() {
		<-sigCh
		printPeaks(*mode)
		os.Exit(0)
	}()

	doMlockall()

	sourceDir := "/proc/pressure"
	cpuFile := "/proc/pressure/cpu"
	ioFile := "/proc/pressure/io"
	memoryFile := "/proc/pressure/memory"

	if *target != "SYSTEM_WIDE" {
		mountPoint := findCGroup2Mount()
		if mountPoint == "" {
			fmt.Fprintln(os.Stderr, "ERROR: cgroup2 not mounted")
			os.Exit(1)
		}
		t := "/" + strings.Trim(*target, "/")
		sourceDir = mountPoint + t
		cpuFile = sourceDir + "/cpu.pressure"
		ioFile = sourceDir + "/io.pressure"
		memoryFile = sourceDir + "/memory.pressure"
	}

	if !suppressOutput {
		fmt.Printf("Starting psi2log, target: %s, mode: %s, interval: %.0f sec\n", *target, *mode, *interval)
	}

	switch *mode {
	case "0":
		runMode0(cpuFile, ioFile, memoryFile, *interval, suppressOutput)
	case "1":
		runMode1(cpuFile, ioFile, memoryFile, *interval, suppressOutput)
	case "2":
		runMode2(cpuFile, ioFile, memoryFile, *interval, suppressOutput)
	}
}

func runMode0(cpuFile, ioFile, memoryFile string, interval float64, suppress bool) {
	if !suppress {
		fmt.Println("================================================")
		fmt.Println("     cpu      ||               io              ||            memory")
		fmt.Println("============= || ============================= || =========================")
		fmt.Println("     some     ||      some     |      full     ||      some     |      full")
		fmt.Println("------------- || ------------- | ------------- || ------------- | ------------")
		fmt.Println(" avg10  avg60 ||  avg10  avg60 |  avg10  avg60 ||  avg10  avg60 |  avg10  avg60")
		fmt.Println("------ ------ || ------ ------ | ------ ------ || ------ ------ | ------ ------")
	}

	for {
		cpu, _ := proc.ReadPSIValue(cpuFile, "some_avg10")
		cpu60, _ := proc.ReadPSIValue(cpuFile, "some_avg60")
		io10, _ := proc.ReadPSIValue(ioFile, "some_avg10")
		io60, _ := proc.ReadPSIValue(ioFile, "some_avg60")
		iof10, _ := proc.ReadPSIValue(ioFile, "full_avg10")
		iof60, _ := proc.ReadPSIValue(ioFile, "full_avg60")
		m10, _ := proc.ReadPSIValue(memoryFile, "some_avg10")
		m60, _ := proc.ReadPSIValue(memoryFile, "some_avg60")
		mf10, _ := proc.ReadPSIValue(memoryFile, "full_avg10")
		mf60, _ := proc.ReadPSIValue(memoryFile, "full_avg60")

		if !suppress {
			fmt.Printf("%6.2f %6.2f || %6.2f %6.2f | %6.2f %6.2f || %6.2f %6.2f | %6.2f %6.2f\n",
				cpu, cpu60, io10, io60, iof10, iof60, m10, m60, mf10, mf60)
		}

		updatePeak("c_some_avg10", cpu)
		updatePeak("c_some_avg60", cpu60)
		updatePeak("i_some_avg10", io10)
		updatePeak("i_some_avg60", io60)
		updatePeak("i_full_avg10", iof10)
		updatePeak("i_full_avg60", iof60)
		updatePeak("m_some_avg10", m10)
		updatePeak("m_some_avg60", m60)
		updatePeak("m_full_avg10", mf10)
		updatePeak("m_full_avg60", mf60)

		time.Sleep(time.Duration(interval * float64(time.Second)))
	}
}

func runMode1(cpuFile, ioFile, memoryFile string, interval float64, suppress bool) {
	if !suppress {
		fmt.Println("==============================================================")
		fmt.Println("        cpu          ||                     io                ||                   memory")
		fmt.Println("==================== || ===================================== || =================================")
		fmt.Println("        some         ||         some         |         full   ||         some         |         full")
		fmt.Println("-------------------- || -------------------- | ---------------- || -------------------- | ----------------")
		fmt.Println(" avg10  avg60 avg300 ||  avg10  avg60 avg300 |  avg10  avg60 avg300 ||  avg10  avg60 avg300 |  avg10  avg60 avg300")
		fmt.Println("------ ------ ------ || ------ ------ ------ | ------ ------ ------ || ------ ------ ------ | ------ ------ ------")
	}

	for {
		cs10, _ := proc.ReadPSIValue(cpuFile, "some_avg10")
		cs60, _ := proc.ReadPSIValue(cpuFile, "some_avg60")
		cs300, _ := proc.ReadPSIValue(cpuFile, "some_avg300")
		is10, _ := proc.ReadPSIValue(ioFile, "some_avg10")
		is60, _ := proc.ReadPSIValue(ioFile, "some_avg60")
		is300, _ := proc.ReadPSIValue(ioFile, "some_avg300")
		isf10, _ := proc.ReadPSIValue(ioFile, "full_avg10")
		isf60, _ := proc.ReadPSIValue(ioFile, "full_avg60")
		isf300, _ := proc.ReadPSIValue(ioFile, "full_avg300")
		ms10, _ := proc.ReadPSIValue(memoryFile, "some_avg10")
		ms60, _ := proc.ReadPSIValue(memoryFile, "some_avg60")
		ms300, _ := proc.ReadPSIValue(memoryFile, "some_avg300")
		msf10, _ := proc.ReadPSIValue(memoryFile, "full_avg10")
		msf60, _ := proc.ReadPSIValue(memoryFile, "full_avg60")
		msf300, _ := proc.ReadPSIValue(memoryFile, "full_avg300")

		if !suppress {
			fmt.Printf("%6.2f %6.2f %6.2f || %6.2f %6.2f %6.2f | %6.2f %6.2f %6.2f || %6.2f %6.2f %6.2f | %6.2f %6.2f %6.2f\n",
				cs10, cs60, cs300, is10, is60, is300, isf10, isf60, isf300, ms10, ms60, ms300, msf10, msf60, msf300)
		}

		updatePeak("c_some_avg10", cs10)
		updatePeak("c_some_avg60", cs60)
		updatePeak("c_some_avg300", cs300)
		updatePeak("i_some_avg10", is10)
		updatePeak("i_some_avg60", is60)
		updatePeak("i_some_avg300", is300)
		updatePeak("i_full_avg10", isf10)
		updatePeak("i_full_avg60", isf60)
		updatePeak("i_full_avg300", isf300)
		updatePeak("m_some_avg10", ms10)
		updatePeak("m_some_avg60", ms60)
		updatePeak("m_some_avg300", ms300)
		updatePeak("m_full_avg10", msf10)
		updatePeak("m_full_avg60", msf60)
		updatePeak("m_full_avg300", msf300)

		time.Sleep(time.Duration(interval * float64(time.Second)))
	}
}

func runMode2(cpuFile, ioFile, memoryFile string, interval float64, suppress bool) {
	if !suppress {
		fmt.Println("----- - ----------- - ----------- -")
		fmt.Println(" cpu  |      io     |    memory   |")
		fmt.Println("----- | ----------- | ----------- |")
		fmt.Println(" some |  some  full |  some  full | interval")
		fmt.Println("----- | ----- ----- | ----- ----- | --------")
	}

	t0 := time.Now()
	prevCPU, _ := readCPUTotal(cpuFile)
	prevIOS, prevIOF, _ := readMemTotal(ioFile)
	prevMS, prevMF, _ := readMemTotal(memoryFile)
	time.Sleep(time.Duration(interval * float64(time.Second)))

	for {
		t1 := time.Now()
		currCPU, _ := readCPUTotal(cpuFile)
		currIOS, currIOF, _ := readMemTotal(ioFile)
		currMS, currMF, _ := readMemTotal(memoryFile)

		d := t1.Sub(t0).Seconds()
		dtCPU := (currCPU - prevCPU) / d / 10000
		dtIOS := (currIOS - prevIOS) / d / 10000
		dtIOF := (currIOF - prevIOF) / d / 10000
		dtMS := (currMS - prevMS) / d / 10000
		dtMF := (currMF - prevMF) / d / 10000

		updatePeak("avg_cs", dtCPU)
		updatePeak("avg_is", dtIOS)
		updatePeak("avg_if", dtIOF)
		updatePeak("avg_ms", dtMS)
		updatePeak("avg_mf", dtMF)

		if !suppress {
			fmt.Printf("%5.1f | %5.1f %5.1f | %5.1f %5.1f | %.2f\n", dtCPU, dtIOS, dtIOF, dtMS, dtMF, d)
		}

		prevCPU = currCPU
		prevIOS = currIOS
		prevIOF = currIOF
		prevMS = currMS
		prevMF = currMF
		t0 = t1

		time.Sleep(time.Duration(interval * float64(time.Second)))
	}
}

func readCPUTotal(path string) (float64, error) {
	m, err := proc.ReadPSIFile(path)
	if err != nil {
		return 0, err
	}
	return m.SomeAvg10, nil
}

func readMemTotal(path string) (someTotal, fullTotal float64, err error) {
	m, err := proc.ReadPSIFile(path)
	if err != nil {
		return 0, 0, err
	}
	return m.SomeAvg10, m.FullAvg10, nil
}

func updatePeak(key string, val float64) {
	if curr, ok := peaks[key]; !ok || val > curr {
		peaks[key] = val
	}
}

func printPeaks(mode string) {
	fmt.Println("\nPeak values:")
	for k, v := range peaks {
		fmt.Printf("  %s: %.2f\n", k, v)
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

func doMlockall() {
	syscall.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE)
}
