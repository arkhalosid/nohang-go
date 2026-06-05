package main

import (
	"flag"
	"fmt"
	"github.com/user/nohang/internal/action"
	"github.com/user/nohang/internal/badness"
	"github.com/user/nohang/internal/config"
	"github.com/user/nohang/internal/kmsg"
	"github.com/user/nohang/internal/monitor"
	"github.com/user/nohang/internal/notifier"
	"github.com/user/nohang/internal/proc"
	"github.com/user/nohang/internal/stats"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var version = "0.3.0"
var startTime = time.Now()

func main() {
	var configPath string
	help := flag.Bool("h", false, "show help")
	showVersion := flag.Bool("v", false, "show version")
	memload := flag.Bool("m", false, "consume memory")
	flag.StringVar(&configPath, "c", "", "path to config (alias for --config)")
	flag.StringVar(&configPath, "config", "", "path to the config file")
	checkCfg := flag.Bool("check", false, "check config")
	monitorMode := flag.Bool("monitor", false, "start monitoring")
	tasksMode := flag.Bool("tasks", false, "show tasks")
	flag.Parse()

	if *help || len(os.Args) == 1 {
		printHelp()
		return
	}
	if *showVersion {
		fmt.Printf("nohang %s\n", version)
		return
	}
	if *memload {
		memloadFn()
		return
	}
	if configPath == "" {
		fmt.Fprintln(os.Stderr, "ERROR: -c/--config is required with --monitor, --check, --tasks")
		os.Exit(1)
	}

	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	if err := validateConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	if *checkCfg {
		cfg.Print(func(format string, args ...interface{}) {
			fmt.Printf(format+"\n", args...)
		})
		fmt.Println("\nconfig is OK")
		return
	}

	if *tasksMode {
		printTasks(cfg)
		return
	}

	if *monitorMode {
		runDaemon(cfg)
	}
}

func printHelp() {
	fmt.Print(`usage: nohang [-h|--help] [-v|--version] [-m|--memload]
              [-c|--config CONFIG] [--check] [--monitor] [--tasks]

optional arguments:
  -h, --help            show this help message and exit
  -v, --version         show version of installed package and exit
  -m, --memload         consume memory until 40 MiB (MemAvailable + SwapFree)
                        remain free, and terminate the process
  -c CONFIG, --config CONFIG
                        path to the config file. This should only be used
                        with one of the following options:
                        --monitor, --tasks, --check
  --check               check and show the configuration and exit. This should
                        only be used with -c/--config CONFIG option
  --monitor             start monitoring. This should only be used with
                        -c/--config CONFIG option
  --tasks               show tasks state and exit. This should only be used
                        with -c/--config CONFIG option
`)
}

func log(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func validateConfig(cfg *config.Config) error {
	mi, err := proc.GetMemInfo()
	if err != nil {
		return fmt.Errorf("cannot read /proc/meminfo: %w", err)
	}
	memTotal := mi.MemTotal
	if memTotal == 0 {
		return fmt.Errorf("cannot determine total memory")
	}

	if cfg.SoftThresholdMinMem != "" {
		kb, mb, pct, err := config.CalculatePercent(cfg.SoftThresholdMinMem, memTotal)
		if err != nil {
			return fmt.Errorf("soft_threshold_min_mem: %w", err)
		}
		if kb > memTotal*0.5 || kb < 0 {
			return fmt.Errorf("soft_threshold_min_mem: invalid value")
		}
		cfg.SoftThresholdMinMemKB = kb
		cfg.SoftThresholdMinMemMB = mb
		cfg.SoftThresholdMinMemPct = pct
	}
	if cfg.HardThresholdMinMem != "" {
		kb, mb, pct, err := config.CalculatePercent(cfg.HardThresholdMinMem, memTotal)
		if err != nil {
			return fmt.Errorf("hard_threshold_min_mem: %w", err)
		}
		if kb > memTotal*0.5 || kb < 0 {
			return fmt.Errorf("hard_threshold_min_mem: invalid value")
		}
		cfg.HardThresholdMinMemKB = kb
		cfg.HardThresholdMinMemMB = mb
		cfg.HardThresholdMinMemPct = pct
	}
	if cfg.WarningThresholdMinMem != "" {
		kb, mb, pct, err := config.CalculatePercent(cfg.WarningThresholdMinMem, memTotal)
		if err != nil {
			return fmt.Errorf("warning_threshold_min_mem: %w", err)
		}
		if kb > memTotal || kb < 0 {
			return fmt.Errorf("warning_threshold_min_mem: invalid value")
		}
		cfg.WarningThresholdMinMemKB = kb
		cfg.WarningThresholdMinMemMB = mb
		cfg.WarningThresholdMinMemPct = pct
	}

	if cfg.SoftThresholdMaxZRAM != "" {
		kb, mb, pct, err := config.CalculatePercent(cfg.SoftThresholdMaxZRAM, memTotal)
		if err != nil {
			return fmt.Errorf("soft_threshold_max_zram: %w", err)
		}
		if kb > memTotal*0.9 || kb < memTotal*0.1 {
			return fmt.Errorf("soft_threshold_max_zram: invalid value")
		}
		cfg.SoftThresholdMaxZRAMKB = kb
		cfg.SoftThresholdMaxZRAMMB = mb
		cfg.SoftThresholdMaxZRAMPct = pct
	}
	if cfg.HardThresholdMaxZRAM != "" {
		kb, mb, pct, err := config.CalculatePercent(cfg.HardThresholdMaxZRAM, memTotal)
		if err != nil {
			return fmt.Errorf("hard_threshold_max_zram: %w", err)
		}
		if kb > memTotal*0.9 || kb < memTotal*0.1 {
			return fmt.Errorf("hard_threshold_max_zram: invalid value")
		}
		cfg.HardThresholdMaxZRAMKB = kb
		cfg.HardThresholdMaxZRAMMB = mb
		cfg.HardThresholdMaxZRAMPct = pct
	}
	if cfg.WarningThresholdMaxZRAM != "" {
		kb, mb, pct, err := config.CalculatePercent(cfg.WarningThresholdMaxZRAM, memTotal)
		if err != nil {
			return fmt.Errorf("warning_threshold_max_zram: %w", err)
		}
		if kb > memTotal || kb < 0 {
			return fmt.Errorf("warning_threshold_max_zram: invalid value")
		}
		cfg.WarningThresholdMaxZRAMKB = kb
		cfg.WarningThresholdMaxZRAMMB = mb
		cfg.WarningThresholdMaxZRAMPct = pct
	}

	if cfg.SoftThresholdMinSwap != "" {
		kb, pct, isPct, err := config.ParseSwapThreshold(cfg.SoftThresholdMinSwap)
		if err != nil {
			return fmt.Errorf("soft_threshold_min_swap: %w", err)
		}
		cfg.SoftSwapIsPercent = isPct
		if isPct {
			cfg.SoftThresholdMinSwapPct = pct
		} else {
			cfg.SoftThresholdMinSwapKB = kb
		}
	}
	if cfg.HardThresholdMinSwap != "" {
		kb, pct, isPct, err := config.ParseSwapThreshold(cfg.HardThresholdMinSwap)
		if err != nil {
			return fmt.Errorf("hard_threshold_min_swap: %w", err)
		}
		cfg.HardSwapIsPercent = isPct
		if isPct {
			cfg.HardThresholdMinSwapPct = pct
		} else {
			cfg.HardThresholdMinSwapKB = kb
		}
	}
	if cfg.WarningThresholdMinSwap != "" {
		kb, pct, isPct, err := config.ParseSwapThreshold(cfg.WarningThresholdMinSwap)
		if err != nil {
			return fmt.Errorf("warning_threshold_min_swap: %w", err)
		}
		cfg.WarningSwapIsPercent = isPct
		if isPct {
			cfg.WarningThresholdMinSwapPct = pct
		} else {
			cfg.WarningThresholdMinSwapKB = kb
		}
	}

	return nil
}

func printTasks(cfg *config.Config) {
	selfPID := os.Getpid()
	result := badness.FindVictim(selfPID, cfg)
	if result != nil {
		fmt.Printf("Process with highest badness: PID=%d, badness=%d\n", result.PID, result.Badness)
	}
}

func runDaemon(cfg *config.Config) {
	selfPID := os.Getpid()
	victimCache := action.NewVictimCache(cfg.VictimCacheTime)
	stat := stats.New(cfg.PrintStatistics)
	lastActionTime := time.Now()

	log("Starting nohang with config %s", os.Args[2])

	if cfg.PrintConfigAtStartup {
		cfg.Print(func(format string, args ...interface{}) {
			log(format, args...)
		})
	}

	if err := mlockall(); err != nil {
		log("WARNING: cannot lock process memory: %v", err)
	}

	notif := notifier.New(cfg.EnvCacheTime, cfg.DebugGUINotifications, log)

	var kmsgMon *kmsg.Monitor
	if cfg.CheckKmsg {
		kmsgMon = kmsg.New(cfg.DebugKmsg, log, stat.Update, func() { stat.Print(log, "") })
		if kmsgMon.IsKMsgOK() {
			go kmsgMon.Run()
		}
	}

	psiMon := monitor.NewPSIMonitor()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP)

	warnTimer := time.Now()
	lastMemReport := time.Now()

	log("Monitoring has started!")

	for {
		select {
		case sig := <-sigCh:
			log("Got the %s signal", sig)
			stat.Print(log, time.Since(startTime).String())
			mi, _ := proc.GetMemInfo()
			if mi != nil {
				log("MemAvailable: %.0f MiB, SwapFree: %.0f MiB", mi.MemAvailable/1024, mi.SwapFree/1024)
			}
			return
		default:
		}

		memResult := monitor.CheckMemSwap(cfg, 0)
		if memResult == nil {
			sleepFn(cfg, 0, 0, 0, log)
			continue
		}

		var zramResult *monitor.ThresholdResult
		if cfg.ZRAMCheckingEnabled {
			zramResult = monitor.CheckZRAM(cfg, memResult.MemAvail, 0)
		}

		lastOOM := time.Time{}
		if kmsgMon != nil {
			lastOOM = kmsgMon.GetLastOOMTime()
		}
		actionTime := lastActionTime
		if !lastOOM.IsZero() && lastOOM.After(actionTime) {
			actionTime = lastOOM
		}
		psiResult := psiMon.Check(cfg, memResult.MemAvail, actionTime)

		// Memory check reporting
		if cfg.PrintMemCheckResults {
			now := time.Now()
			if now.Sub(lastMemReport).Seconds() >= cfg.MinMemReportInterval {
				mi, _ := proc.GetMemInfo()
				if mi != nil {
					log("MemAvail: %.0f M, %.1f %% | SwapFree: %.0f M, %.1f %%",
						mi.MemAvailable/1024, mi.MemAvailable/mi.MemTotal*100,
						mi.SwapFree/1024, mi.SwapFree/(mi.SwapTotal+0.1)*100)
				}
				lastMemReport = now
			}
		}

		threshold := resolveThreshold(memResult.Signal, zramResult, psiResult)

		if threshold == monitor.SigKill || threshold == monitor.SigTerm {
			sig := monitor.SigTerm
			if threshold == monitor.SigKill {
				sig = monitor.SigKill
			}
			action.ApplyCorrectiveAction(
				sig, cfg, selfPID, victimCache, &lastActionTime, notif,
				log, stat.Update, func() { stat.Print(log, "") },
				func(d time.Duration) { time.Sleep(d) },
				time.Duration(cfg.MinSleep*float64(time.Second)),
			)
			continue
		}

		if cfg.LowMemoryWarningsEnabled {
			if threshold == monitor.SigWarn {
				if time.Since(warnTimer).Seconds() > cfg.MinPostWarningDelay {
					sendWarning(cfg, notif, log, stat)
					warnTimer = time.Now()
				}
			}
		}

		zramUsed := 0.0
		if zramResult != nil {
			zramUsed = zramResult.MemUsedZRAM
		}
		sleepDuration := monitor.CalcSleep(cfg, memResult.MemAvail, memResult.SwapFree,
			zramUsed, log)
		if cfg.DebugSleep {
			log("Sleep %.2fs", sleepDuration.Seconds())
		}
		time.Sleep(sleepDuration)
	}
}

func resolveThreshold(memSig int, zramResult, psiResult *monitor.ThresholdResult) int {
	if memSig == monitor.SigKill {
		return monitor.SigKill
	}
	if zramResult != nil && zramResult.Signal == monitor.SigKill {
		return monitor.SigKill
	}
	if psiResult != nil && psiResult.Signal == monitor.SigKill {
		return monitor.SigKill
	}
	if memSig == monitor.SigTerm {
		return monitor.SigTerm
	}
	if zramResult != nil && zramResult.Signal == monitor.SigTerm {
		return monitor.SigTerm
	}
	if psiResult != nil && psiResult.Signal == monitor.SigTerm {
		return monitor.SigTerm
	}
	if memSig == monitor.SigWarn {
		return monitor.SigWarn
	}
	if zramResult != nil && zramResult.Signal == monitor.SigWarn {
		return monitor.SigWarn
	}
	if psiResult != nil && psiResult.Signal == monitor.SigWarn {
		return monitor.SigWarn
	}
	return 0
}

func sendWarning(cfg *config.Config, notif *notifier.Notifier, logFn func(string, ...interface{}), stat *stats.Stats) {
	logFn("Warning threshold exceeded")

	if cfg.WarningExe != "" {
		go action.ExecCommandFn(cfg.WarningExe, cfg.ExeTimeout, logFn)
	} else {
		mi, _ := proc.GetMemInfo()
		title := "Low memory"
		var body string
		if mi != nil {
			shPct := mi.Shmem / mi.MemTotal
			if shPct > 0.6 {
				body = fmt.Sprintf("Save your unsaved data!\nClear tmpfs! Shmem: %.0f%%", shPct*100)
			} else if shPct > 0.3 {
				body = fmt.Sprintf("Save your unsaved data!\nClose unused apps!\nClear tmpfs! Shmem: %.0f%%", shPct*100)
			} else {
				body = "Save your unsaved data!\nClose unused apps!"
			}
		} else {
			body = "Save your unsaved data!\nClose unused apps!"
		}
		notif.Send(title, body)
	}
}

func sleepFn(cfg *config.Config, memAvail, swapFree, memUsedZRAM float64, logFn func(string, ...interface{})) {
	d := monitor.CalcSleep(cfg, memAvail, swapFree, memUsedZRAM, logFn)
	if cfg.DebugSleep {
		logFn("Sleep %.2fs", d.Seconds())
	}
	time.Sleep(d)
}

func mlockall() error {
	return syscall.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE)
}

func memloadFn() {
	fmt.Println("memload not implemented in Go port")
	os.Exit(0)
}
