package monitor

import (
	"math"
	"github.com/user/nohang/internal/config"
	"github.com/user/nohang/internal/proc"
	"time"
)

type ThresholdResult struct {
	Signal      int // 0=none, 1=warn, 15=SIGTERM, 9=SIGKILL
	MemInfo     string
	MemAvail    float64
	SwapFree    float64
	SwapTotal   float64
	MemUsedZRAM float64
}

const (
	SigWarn   = 1
	SigTerm   = 15
	SigKill   = 9
)

func CheckMemSwap(cfg *config.Config, memTotal float64) *ThresholdResult {
	mi, err := proc.GetMemInfo()
	if err != nil {
		return nil
	}
	memAvail := mi.MemAvailable
	swapTotal := mi.SwapTotal
	swapFree := mi.SwapFree

	hardSwapKB := cfg.HardThresholdMinSwapKB
	if cfg.HardSwapIsPercent {
		hardSwapKB = swapTotal * cfg.HardThresholdMinSwapPct / 100.0
	}
	softSwapKB := cfg.SoftThresholdMinSwapKB
	if cfg.SoftSwapIsPercent {
		softSwapKB = swapTotal * cfg.SoftThresholdMinSwapPct / 100.0
	}
	warnSwapKB := cfg.WarningThresholdMinSwapKB
	if cfg.WarningSwapIsPercent {
		warnSwapKB = swapTotal * cfg.WarningThresholdMinSwapPct / 100.0
	}

	r := &ThresholdResult{
		MemAvail:  memAvail,
		SwapFree:  swapFree,
		SwapTotal: swapTotal,
	}

	if memAvail <= cfg.HardThresholdMinMemKB && swapFree <= hardSwapKB {
		r.Signal = SigKill
		r.MemInfo = formatMemInfo("hard", memAvail, cfg.HardThresholdMinMemKB,
			cfg.HardThresholdMinMemPct, swapFree, hardSwapKB, swapTotal)
		return r
	}
	if memAvail <= cfg.SoftThresholdMinMemKB && swapFree <= softSwapKB {
		r.Signal = SigTerm
		r.MemInfo = formatMemInfo("soft", memAvail, cfg.SoftThresholdMinMemKB,
			cfg.SoftThresholdMinMemPct, swapFree, softSwapKB, swapTotal)
		return r
	}
	if cfg.LowMemoryWarningsEnabled {
		if memAvail <= cfg.WarningThresholdMinMemKB && swapFree <= warnSwapKB {
			r.Signal = SigWarn
			return r
		}
	}

	return r
}

func CheckZRAM(cfg *config.Config, memAvail float64, memTotal float64) *ThresholdResult {
	memUsedZRAM := proc.CheckZRAM()
	r := &ThresholdResult{MemUsedZRAM: memUsedZRAM}

	maHard := memAvail <= cfg.HardThresholdMinMemKB
	maSoft := memAvail <= cfg.SoftThresholdMinMemKB
	maWarn := memAvail <= cfg.WarningThresholdMinMemKB

	if memUsedZRAM >= cfg.HardThresholdMaxZRAMKB && maHard {
		r.Signal = SigKill
		r.MemInfo = formatZRAMInfo("hard", memAvail, cfg.HardThresholdMinMemKB,
			cfg.HardThresholdMinMemPct, memUsedZRAM, cfg.HardThresholdMaxZRAMKB,
			cfg.HardThresholdMaxZRAMPct, memTotal)
		return r
	}
	if memUsedZRAM >= cfg.SoftThresholdMaxZRAMKB && maSoft {
		r.Signal = SigTerm
		r.MemInfo = formatZRAMInfo("soft", memAvail, cfg.SoftThresholdMinMemKB,
			cfg.SoftThresholdMinMemPct, memUsedZRAM, cfg.SoftThresholdMaxZRAMKB,
			cfg.SoftThresholdMaxZRAMPct, memTotal)
		return r
	}
	if cfg.LowMemoryWarningsEnabled {
		if memUsedZRAM >= cfg.WarningThresholdMaxZRAMKB && maWarn {
			r.Signal = SigWarn
			return r
		}
	}
	return r
}

type PSIMonitor struct {
	killExceededTimer float64
	termExceededTimer float64
	lastCheckTime     time.Time
}

func NewPSIMonitor() *PSIMonitor {
	return &PSIMonitor{
		killExceededTimer: -0.0001,
		termExceededTimer: -0.0001,
		lastCheckTime:     time.Now(),
	}
}

func (pm *PSIMonitor) Check(cfg *config.Config, memAvail float64, lastActionTime time.Time) *ThresholdResult {
	r := &ThresholdResult{}

	maHard := memAvail <= cfg.HardThresholdMinMemKB
	maSoft := memAvail <= cfg.SoftThresholdMinMemKB
	maWarn := memAvail <= cfg.WarningThresholdMinMemKB

	if (!maHard && !maSoft && !maWarn) || cfg.SoftThresholdMinSwap == "" {
		return r
	}

	now := time.Now()
	delta := now.Sub(pm.lastCheckTime).Seconds()
	pm.lastCheckTime = now

	psiVal, err := proc.ReadPSIValue(cfg.PSIPath, cfg.PSIMetrics)
	if err != nil {
		return r
	}

	postActionDelay := now.Sub(lastActionTime).Seconds()

	if psiVal >= cfg.HardThresholdMaxPSI && maHard {
		if pm.killExceededTimer < 0 {
			pm.killExceededTimer = 0
		} else {
			pm.killExceededTimer += delta
		}
	} else {
		pm.killExceededTimer = -0.0001
	}

	if pm.killExceededTimer >= cfg.PSIExcessDuration &&
		postActionDelay >= cfg.PSIPostActionDelay && maHard {
		r.Signal = SigKill
		r.MemInfo = formatPSIInfo("hard", memAvail, cfg.HardThresholdMinMemKB,
			cfg.HardThresholdMinMemPct, psiVal, cfg.HardThresholdMaxPSI,
			cfg.PSIExcessDuration, pm.killExceededTimer)
		return r
	}

	if psiVal >= cfg.SoftThresholdMaxPSI && maSoft {
		if pm.termExceededTimer < 0 {
			pm.termExceededTimer = 0
		} else {
			pm.termExceededTimer += delta
		}
	} else {
		pm.termExceededTimer = -0.0001
	}

	if pm.termExceededTimer >= cfg.PSIExcessDuration &&
		postActionDelay >= cfg.PSIPostActionDelay && maSoft {
		r.Signal = SigTerm
		r.MemInfo = formatPSIInfo("soft", memAvail, cfg.SoftThresholdMinMemKB,
			cfg.SoftThresholdMinMemPct, psiVal, cfg.SoftThresholdMaxPSI,
			cfg.PSIExcessDuration, pm.termExceededTimer)
		return r
	}

	if cfg.LowMemoryWarningsEnabled && psiVal >= cfg.WarningThresholdMaxPSI && maWarn {
		r.Signal = SigWarn
	}

	return r
}

func formatMemInfo(level string, memAvail, memThresh, memPct, swapFree, swapThresh, swapTotal float64) string {
	return ""
}

func formatZRAMInfo(level string, memAvail, memThresh, memPct, zramUsed, zramThresh, zramPct, memTotal float64) string {
	return ""
}

func formatPSIInfo(level string, memAvail, memThresh, memPct, psiVal, psiThresh, excessDur, excTimer float64) string {
	return ""
}

func CalcSleep(cfg *config.Config, memAvail, swapFree, memUsedZRAM float64, log func(string, ...interface{})) time.Duration {
	if cfg.MaxSleep == cfg.MinSleep {
		return time.Duration(cfg.MinSleep * float64(time.Second))
	}

	memPoint := memAvail - cfg.SoftThresholdMinMemKB
	if cfg.HardThresholdMinMemKB < cfg.SoftThresholdMinMemKB {
		memPoint = memAvail - cfg.HardThresholdMinMemKB
	}
	if memPoint < 0 {
		memPoint = 0
	}

	swapPoint := swapFree - cfg.SoftThresholdMinSwapKB
	if cfg.HardThresholdMinSwapKB < cfg.SoftThresholdMinSwapKB {
		swapPoint = swapFree - cfg.HardThresholdMinSwapKB
	}
	if swapPoint < 0 {
		swapPoint = 0
	}

	tMem := memPoint / cfg.FillRateMem
	tSwap := swapPoint / cfg.FillRateSwap
	t := tMem + tSwap

	if cfg.ZRAMCheckingEnabled {
		tZRAM := (memAvail*0.8 - memUsedZRAM) / cfg.FillRateZRAM
		if tZRAM < 0 {
			tZRAM = 0
		}
		if tMem+tSwap < tMem+tZRAM {
			t = tMem + tSwap
		} else {
			t = t
		}
	}

	if t > cfg.MaxSleep {
		t = cfg.MaxSleep
	}
	if t < cfg.MinSleep {
		t = cfg.MinSleep
	}

	return time.Duration(t * float64(time.Second))
}

func KibToMib(kb float64) float64 {
	return math.Round(kb / 1024)
}
