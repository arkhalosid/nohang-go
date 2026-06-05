package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type BadnessAdj struct {
	Adjustment int
	Regexp     *regexp.Regexp
}

type SoftAction struct {
	Unit    string // "name", "cgroup_v1", "cgroup_v2"
	Regexp  *regexp.Regexp
	Command string
}

type Config struct {
	SeparateLog                bool
	DebugPSI                   bool
	PrintStatistics            bool
	PrintProcTable             bool
	PrintVictimStatus          bool
	PrintVictimCmdline         bool
	PrintConfigAtStartup       bool
	PrintMemCheckResults       bool
	DebugSleep                 bool
	HideCorrectiveActionType   bool
	LowMemoryWarningsEnabled   bool
	PostActionGUINotifications bool
	DebugThreading             bool
	PSICheckingEnabled         bool
	ZRAMCheckingEnabled        bool
	DebugGUINotifications      bool
	IgnorePositiveOOMScoreAdj  bool

	SoftThresholdMinMem     string
	HardThresholdMinMem     string
	WarningThresholdMinMem  string
	SoftThresholdMaxZRAM    string
	HardThresholdMaxZRAM    string
	WarningThresholdMaxZRAM string

	SoftThresholdMinMemKB     float64
	SoftThresholdMinMemMB     float64
	SoftThresholdMinMemPct    float64
	HardThresholdMinMemKB     float64
	HardThresholdMinMemMB     float64
	HardThresholdMinMemPct    float64
	WarningThresholdMinMemKB  float64
	WarningThresholdMinMemMB  float64
	WarningThresholdMinMemPct float64
	SoftThresholdMaxZRAMKB    float64
	SoftThresholdMaxZRAMMB    float64
	SoftThresholdMaxZRAMPct   float64
	HardThresholdMaxZRAMKB    float64
	HardThresholdMaxZRAMMB    float64
	HardThresholdMaxZRAMPct   float64
	WarningThresholdMaxZRAMKB  float64
	WarningThresholdMaxZRAMMB  float64
	WarningThresholdMaxZRAMPct float64

	SoftThresholdMinSwap    string
	HardThresholdMinSwap    string
	WarningThresholdMinSwap string
	SoftThresholdMinSwapKB  float64
	SoftThresholdMinSwapPct float64
	SoftSwapIsPercent       bool
	HardThresholdMinSwapKB  float64
	HardThresholdMinSwapPct float64
	HardSwapIsPercent       bool
	WarningThresholdMinSwapKB  float64
	WarningThresholdMinSwapPct float64
	WarningSwapIsPercent       bool

	PostZombieDelay       float64
	VictimCacheTime       float64
	EnvCacheTime          float64
	ExeTimeout            float64
	FillRateMem           float64
	FillRateSwap          float64
	FillRateZRAM          float64
	PostSoftActionDelay   float64
	PSIPostActionDelay    float64
	HardThresholdMaxPSI   float64
	SoftThresholdMaxPSI   float64
	WarningThresholdMaxPSI float64
	MinBadness            int
	MinPostWarningDelay   float64
	MaxVictimAncestryDepth int
	MaxSoftExitTime       float64
	PostKillExe           string
	PSIPath               string
	PSIMetrics            string
	WarningExe            string
	ExtraTableInfo        string
	MinMemReportInterval  float64
	PSIExcessDuration     float64
	MaxSleep              float64
	MinSleep              float64

	CheckKmsg bool
	DebugKmsg bool

	BadnessAdjReName      []BadnessAdj
	BadnessAdjReCmdline   []BadnessAdj
	BadnessAdjReUID       []BadnessAdj
	BadnessAdjReCgroupV1  []BadnessAdj
	BadnessAdjReCgroupV2  []BadnessAdj
	BadnessAdjReRealpath  []BadnessAdj
	BadnessAdjReCwd       []BadnessAdj
	BadnessAdjReEnviron   []BadnessAdj
	SoftActions           []SoftAction

	RegexMatching      bool
	ReMatchName      bool
	ReMatchCmdline   bool
	ReMatchUID       bool
	ReMatchCgroupV1  bool
	ReMatchCgroupV2  bool
	ReMatchRealpath  bool
	ReMatchCwd       bool
	ReMatchEnviron   bool
}

func ParseConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	cfg := &Config{}
	kv := make(map[string]string)

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r\n")
		if line == "" || line[0] == '#' || line[0] == ' ' || line[0] == '\t' {
			continue
		}

		switch {
		case strings.HasPrefix(line, "@check_kmsg"):
			cfg.CheckKmsg = true
		case strings.HasPrefix(line, "@debug_kmsg"):
			cfg.DebugKmsg = true
		case strings.HasPrefix(line, "@SOFT_ACTION_RE_NAME"):
			sa := parseSoftAction(line, "@SOFT_ACTION_RE_NAME", "name")
			if sa != nil {
				cfg.SoftActions = append(cfg.SoftActions, *sa)
			}
		case strings.HasPrefix(line, "@SOFT_ACTION_RE_CGROUP_V1"):
			sa := parseSoftAction(line, "@SOFT_ACTION_RE_CGROUP_V1", "cgroup_v1")
			if sa != nil {
				cfg.SoftActions = append(cfg.SoftActions, *sa)
			}
		case strings.HasPrefix(line, "@SOFT_ACTION_RE_CGROUP_V2"):
			sa := parseSoftAction(line, "@SOFT_ACTION_RE_CGROUP_V2", "cgroup_v2")
			if sa != nil {
				cfg.SoftActions = append(cfg.SoftActions, *sa)
			}
		case strings.HasPrefix(line, "@BADNESS_ADJ_RE_NAME"):
			ba := parseBadnessAdj(line, "@BADNESS_ADJ_RE_NAME")
			if ba != nil {
				cfg.BadnessAdjReName = append(cfg.BadnessAdjReName, *ba)
			}
		case strings.HasPrefix(line, "@BADNESS_ADJ_RE_CMDLINE"):
			ba := parseBadnessAdj(line, "@BADNESS_ADJ_RE_CMDLINE")
			if ba != nil {
				cfg.BadnessAdjReCmdline = append(cfg.BadnessAdjReCmdline, *ba)
			}
		case strings.HasPrefix(line, "@BADNESS_ADJ_RE_UID"):
			ba := parseBadnessAdj(line, "@BADNESS_ADJ_RE_UID")
			if ba != nil {
				cfg.BadnessAdjReUID = append(cfg.BadnessAdjReUID, *ba)
			}
		case strings.HasPrefix(line, "@BADNESS_ADJ_RE_CGROUP_V1"):
			ba := parseBadnessAdj(line, "@BADNESS_ADJ_RE_CGROUP_V1")
			if ba != nil {
				cfg.BadnessAdjReCgroupV1 = append(cfg.BadnessAdjReCgroupV1, *ba)
			}
		case strings.HasPrefix(line, "@BADNESS_ADJ_RE_CGROUP_V2"):
			ba := parseBadnessAdj(line, "@BADNESS_ADJ_RE_CGROUP_V2")
			if ba != nil {
				cfg.BadnessAdjReCgroupV2 = append(cfg.BadnessAdjReCgroupV2, *ba)
			}
		case strings.HasPrefix(line, "@BADNESS_ADJ_RE_REALPATH"):
			ba := parseBadnessAdj(line, "@BADNESS_ADJ_RE_REALPATH")
			if ba != nil {
				cfg.BadnessAdjReRealpath = append(cfg.BadnessAdjReRealpath, *ba)
			}
		case strings.HasPrefix(line, "@BADNESS_ADJ_RE_CWD"):
			ba := parseBadnessAdj(line, "@BADNESS_ADJ_RE_CWD")
			if ba != nil {
				cfg.BadnessAdjReCwd = append(cfg.BadnessAdjReCwd, *ba)
			}
		case strings.HasPrefix(line, "@BADNESS_ADJ_RE_ENVIRON"):
			ba := parseBadnessAdj(line, "@BADNESS_ADJ_RE_ENVIRON")
			if ba != nil {
				cfg.BadnessAdjReEnviron = append(cfg.BadnessAdjReEnviron, *ba)
			}
		default:
			if strings.Contains(line, "=") && !strings.HasPrefix(line, "@") {
				parts := strings.SplitN(line, "=", 2)
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if _, exists := kv[key]; exists {
					return nil, fmt.Errorf("config key duplication: %s", key)
				}
				kv[key] = val
			}
		}
	}

	cfg.SeparateLog = parseBool(kv, "separate_log")
	cfg.DebugPSI = parseBool(kv, "debug_psi")
	cfg.PrintStatistics = parseBool(kv, "print_statistics")
	cfg.PrintProcTable = parseBool(kv, "print_proc_table")
	cfg.PrintVictimStatus = parseBool(kv, "print_victim_status")
	cfg.PrintVictimCmdline = parseBool(kv, "print_victim_cmdline")
	cfg.PrintConfigAtStartup = parseBool(kv, "print_config_at_startup")
	cfg.PrintMemCheckResults = parseBool(kv, "print_mem_check_results")
	cfg.DebugSleep = parseBool(kv, "debug_sleep")
	cfg.HideCorrectiveActionType = parseBool(kv, "hide_corrective_action_type")
	cfg.LowMemoryWarningsEnabled = parseBool(kv, "low_memory_warnings_enabled")
	cfg.PostActionGUINotifications = parseBool(kv, "post_action_gui_notifications")
	cfg.DebugThreading = parseBool(kv, "debug_threading")
	cfg.PSICheckingEnabled = parseBool(kv, "psi_checking_enabled")
	cfg.ZRAMCheckingEnabled = parseBool(kv, "zram_checking_enabled")
	cfg.DebugGUINotifications = parseBool(kv, "debug_gui_notifications")
	cfg.IgnorePositiveOOMScoreAdj = parseBool(kv, "ignore_positive_oom_score_adj")

	cfg.SoftThresholdMinMem = requireKey(kv, "soft_threshold_min_mem")
	cfg.HardThresholdMinMem = requireKey(kv, "hard_threshold_min_mem")
	cfg.WarningThresholdMinMem = requireKey(kv, "warning_threshold_min_mem")
	cfg.SoftThresholdMaxZRAM = requireKey(kv, "soft_threshold_max_zram")
	cfg.HardThresholdMaxZRAM = requireKey(kv, "hard_threshold_max_zram")
	cfg.WarningThresholdMaxZRAM = requireKey(kv, "warning_threshold_max_zram")

	cfg.SoftThresholdMinSwap = requireKey(kv, "soft_threshold_min_swap")
	cfg.HardThresholdMinSwap = requireKey(kv, "hard_threshold_min_swap")
	cfg.WarningThresholdMinSwap = requireKey(kv, "warning_threshold_min_swap")

	cfg.PostZombieDelay = parseFloat(kv, "post_zombie_delay", true)
	cfg.VictimCacheTime = parseFloat(kv, "victim_cache_time", true)
	cfg.EnvCacheTime = parseFloat(kv, "env_cache_time", true)
	cfg.ExeTimeout = parseFloatMin(kv, "exe_timeout", true, 0.1)
	cfg.FillRateMem = parseFloatMin(kv, "fill_rate_mem", true, 100) * 1024
	cfg.FillRateSwap = parseFloatMin(kv, "fill_rate_swap", true, 100) * 1024
	cfg.FillRateZRAM = parseFloatMin(kv, "fill_rate_zram", true, 100) * 1024
	cfg.PostSoftActionDelay = parseFloatMin(kv, "post_soft_action_delay", true, 0.1)
	cfg.PSIPostActionDelay = parseFloatMin(kv, "psi_post_action_delay", true, 10)
	cfg.HardThresholdMaxPSI = parseFloatRange(kv, "hard_threshold_max_psi", true, 1, 100)
	cfg.SoftThresholdMaxPSI = parseFloatRange(kv, "soft_threshold_max_psi", true, 1, 100)
	cfg.WarningThresholdMaxPSI = parseFloatRange(kv, "warning_threshold_max_psi", true, 1, 100)
	cfg.MinBadness = parseIntMin(kv, "min_badness", true, 1)
	cfg.MinPostWarningDelay = parseFloatMin(kv, "min_post_warning_delay", true, 1)
	cfg.MaxVictimAncestryDepth = parseIntMin(kv, "max_victim_ancestry_depth", true, 1)
	cfg.MaxSoftExitTime = parseFloatMin(kv, "max_soft_exit_time", true, 0.1)

	cfg.PostKillExe = requireKey(kv, "post_kill_exe")
	cfg.PSIPath = requireKey(kv, "psi_path")
	cfg.PSIMetrics = requireKey(kv, "psi_metrics")
	cfg.WarningExe = requireKey(kv, "warning_exe")
	cfg.ExtraTableInfo = requireKey(kv, "extra_table_info")
	cfg.MinMemReportInterval = parseFloatMin(kv, "min_mem_report_interval", true, 0)
	cfg.PSIExcessDuration = parseFloatMin(kv, "psi_excess_duration", true, 0)
	cfg.MaxSleep = parseFloatMin(kv, "max_sleep", true, 0.01)
	cfg.MinSleep = parseFloatMin(kv, "min_sleep", true, 0.01)

	if cfg.MinSleep > cfg.MaxSleep {
		return nil, fmt.Errorf("invalid config: min_sleep > max_sleep")
	}

	cfg.RegexMatching = len(cfg.BadnessAdjReName) > 0
	cfg.ReMatchName = len(cfg.BadnessAdjReName) > 0
	cfg.ReMatchCmdline = len(cfg.BadnessAdjReCmdline) > 0
	cfg.ReMatchUID = len(cfg.BadnessAdjReUID) > 0
	cfg.ReMatchCgroupV1 = len(cfg.BadnessAdjReCgroupV1) > 0
	cfg.ReMatchCgroupV2 = len(cfg.BadnessAdjReCgroupV2) > 0
	cfg.ReMatchRealpath = len(cfg.BadnessAdjReRealpath) > 0
	cfg.ReMatchCwd = len(cfg.BadnessAdjReCwd) > 0
	cfg.ReMatchEnviron = len(cfg.BadnessAdjReEnviron) > 0

	return cfg, nil
}

func parseSoftAction(line, prefix, unit string) *SoftAction {
	rest := strings.TrimPrefix(line, prefix)
	rest = strings.TrimSpace(rest)
	parts := strings.SplitN(rest, "///", 2)
	if len(parts) != 2 {
		return nil
	}
	reStr := strings.TrimSpace(parts[0])
	cmd := strings.TrimSpace(parts[1])
	re, err := regexp.Compile(reStr)
	if err != nil {
		return nil
	}
	return &SoftAction{Unit: unit, Regexp: re, Command: cmd}
}

func parseBadnessAdj(line, prefix string) *BadnessAdj {
	rest := strings.TrimPrefix(line, prefix)
	rest = strings.TrimSpace(rest)
	parts := strings.SplitN(rest, "///", 2)
	if len(parts) != 2 {
		return nil
	}
	adjStr := strings.TrimSpace(parts[0])
	reStr := strings.TrimSpace(parts[1])
	adj, err := strconv.Atoi(adjStr)
	if err != nil {
		return nil
	}
	re, err := regexp.Compile(reStr)
	if err != nil {
		return nil
	}
	return &BadnessAdj{Adjustment: adj, Regexp: re}
}

func parseBool(kv map[string]string, key string) bool {
	v, ok := kv[key]
	if !ok {
		return false
	}
	return v == "True"
}

func requireKey(kv map[string]string, key string) string {
	v, ok := kv[key]
	if !ok {
		return ""
	}
	return v
}

func parseFloat(kv map[string]string, key string, required bool) float64 {
	v, ok := kv[key]
	if !ok {
		if required {
			return 0
		}
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return f
}

func parseFloatMin(kv map[string]string, key string, required bool, min float64) float64 {
	v, ok := kv[key]
	if !ok {
		if required {
			return 0
		}
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < min {
		return 0
	}
	return f
}

func parseFloatRange(kv map[string]string, key string, required bool, min, max float64) float64 {
	v, ok := kv[key]
	if !ok {
		if required {
			return 0
		}
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < min || f > max {
		return 0
	}
	return f
}

func parseIntMin(kv map[string]string, key string, required bool, min int) int {
	v, ok := kv[key]
	if !ok {
		if required {
			return 0
		}
		return 0
	}
	i, err := strconv.Atoi(v)
	if err != nil || i < min {
		return 0
	}
	return i
}

func CalculatePercent(val string, memTotal float64) (kb, mb, pct float64, err error) {
	val = strings.TrimSpace(val)
	if strings.HasSuffix(val, "%") {
		pctStr := strings.TrimRight(strings.TrimSuffix(val, "%"), " \t")
		pct, err = strconv.ParseFloat(pctStr, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid percent value: %s", val)
		}
		kb = pct / 100 * memTotal
		mb = kb / 1024
	} else if strings.HasSuffix(val, "M") {
		mbStr := strings.TrimRight(strings.TrimSuffix(val, "M"), " \t")
		mb, err = strconv.ParseFloat(mbStr, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid MiB value: %s", val)
		}
		kb = mb * 1024
		pct = kb / memTotal * 100
	} else {
		return 0, 0, 0, fmt.Errorf("invalid config value: %s", val)
	}
	return kb, mb, pct, nil
}

func ParseSwapThreshold(val string) (kb float64, pct float64, isPercent bool, err error) {
	val = strings.TrimSpace(val)
	if strings.HasSuffix(val, "%") {
		pctStr := strings.TrimRight(strings.TrimSuffix(val, "%"), " \t")
		pct, err = strconv.ParseFloat(pctStr, 64)
		if err != nil || pct < 0 || pct > 100 {
			return 0, 0, false, fmt.Errorf("invalid swap percent: %s", val)
		}
		return 0, pct, true, nil
	} else if strings.HasSuffix(val, "M") {
		mbStr := strings.TrimRight(strings.TrimSuffix(val, "M"), " \t")
		mb, parseErr := strconv.ParseFloat(mbStr, 64)
		if parseErr != nil || mb < 0 {
			return 0, 0, false, fmt.Errorf("invalid swap MiB: %s", val)
		}
		return mb * 1024, 0, false, nil
	}
	return 0, 0, false, fmt.Errorf("invalid swap value: %s", val)
}

func (c *Config) Print(w func(format string, args ...interface{})) {
	w("0. Check kernel messages for OOM events")
	w("    @check_kmsg:    <%v>", c.CheckKmsg)
	w("    @debug_kmsg:    <%v>", c.DebugKmsg)

	w("\n1. Common zram settings")
	w("    zram_checking_enabled:   %v", c.ZRAMCheckingEnabled)

	w("\n2. Common PSI settings")
	w("    psi_checking_enabled:    %v", c.PSICheckingEnabled)
	w("    psi_path:                %s", c.PSIPath)
	w("    psi_metrics:             %s", c.PSIMetrics)
	w("    psi_excess_duration:     %.0f sec", c.PSIExcessDuration)
	w("    psi_post_action_delay:   %.0f sec", c.PSIPostActionDelay)

	w("\n3. Poll rate")
	w("    fill_rate_mem:   %.0f", c.FillRateMem/1024)
	w("    fill_rate_swap:  %.0f", c.FillRateSwap/1024)
	w("    fill_rate_zram:  %.0f", c.FillRateZRAM/1024)
	w("    max_sleep:       %.0f sec", c.MaxSleep)
	w("    min_sleep:       %.0f sec", c.MinSleep)

	w("\n4. Warnings and notifications")
	w("    post_action_gui_notifications:  %v", c.PostActionGUINotifications)
	w("    hide_corrective_action_type:    %v", c.HideCorrectiveActionType)
	w("    low_memory_warnings_enabled:    %v", c.LowMemoryWarningsEnabled)
	w("    warning_exe:                    %s", c.WarningExe)
	w("    warning_threshold_min_mem:      %.0f MiB, %.1f %%", c.WarningThresholdMinMemMB, c.WarningThresholdMinMemPct)
	w("    warning_threshold_min_swap:     %s", c.WarningThresholdMinSwap)
	w("    warning_threshold_max_zram:     %.0f MiB, %.1f %%", c.WarningThresholdMaxZRAMMB, c.WarningThresholdMaxZRAMPct)
	w("    warning_threshold_max_psi:      %.0f", c.WarningThresholdMaxPSI)
	w("    min_post_warning_delay:         %.0f sec", c.MinPostWarningDelay)
	w("    env_cache_time:                 %.0f", c.EnvCacheTime)

	w("\n5. Soft threshold")
	w("    soft_threshold_min_mem:   %.0f MiB, %.1f %%", c.SoftThresholdMinMemMB, c.SoftThresholdMinMemPct)
	w("    soft_threshold_min_swap:  %s", c.SoftThresholdMinSwap)
	w("    soft_threshold_max_zram:  %.0f MiB, %.1f %%", c.SoftThresholdMaxZRAMMB, c.SoftThresholdMaxZRAMPct)
	w("    soft_threshold_max_psi:   %.0f", c.SoftThresholdMaxPSI)

	w("\n6. Hard threshold")
	w("    hard_threshold_min_mem:   %.0f MiB, %.1f %%", c.HardThresholdMinMemMB, c.HardThresholdMinMemPct)
	w("    hard_threshold_min_swap:  %s", c.HardThresholdMinSwap)
	w("    hard_threshold_max_zram:  %.0f MiB, %.1f %%", c.HardThresholdMaxZRAMMB, c.HardThresholdMaxZRAMPct)
	w("    hard_threshold_max_psi:   %.0f", c.HardThresholdMaxPSI)

	w("\n7. Customize victim selection")
	w("    ignore_positive_oom_score_adj:  %v", c.IgnorePositiveOOMScoreAdj)

	w("\n7.2.1. Matching process names")
	if len(c.BadnessAdjReName) > 0 {
		for _, ba := range c.BadnessAdjReName {
			w("    %12d  %s", ba.Adjustment, ba.Regexp)
		}
	} else {
		w("    (not set)")
	}
}
