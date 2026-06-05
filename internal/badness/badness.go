package badness

import (
	"fmt"
	"github.com/user/nohang/internal/config"
	"github.com/user/nohang/internal/proc"
)

type Result struct {
	PID      int
	Badness  int
	OOMScore int
}

func Calculate(pid int, oomScore int, cfg *config.Config) (int, int) {
	if oomScore == 0 {
		return 0, 0
	}

	badness := oomScore
	oomScoreAdj := -1

	readAdj := func() int {
		if oomScoreAdj < 0 {
			oomScoreAdj = proc.ReadOOMScoreAdj(pid)
		}
		return oomScoreAdj
	}

	if cfg.IgnorePositiveOOMScoreAdj {
		adj := readAdj()
		if adj > 0 {
			badness -= adj * 2 / 3
		}
	}

	if cfg.RegexMatching {
		if cfg.ReMatchName {
			name := proc.ReadComm(pid)
			for _, ba := range cfg.BadnessAdjReName {
				if ba.Regexp.MatchString(name) {
					if ba.Adjustment <= 0 {
						badness += ba.Adjustment
					} else if readAdj() >= 0 {
						badness += ba.Adjustment
					}
				}
			}
		}

		if cfg.ReMatchCgroupV1 {
			cg := proc.ReadCGroup(pid)
			for _, ba := range cfg.BadnessAdjReCgroupV1 {
				if ba.Regexp.MatchString(cg.V1) {
					if ba.Adjustment <= 0 {
						badness += ba.Adjustment
					} else if readAdj() >= 0 {
						badness += ba.Adjustment
					}
				}
			}
		}

		if cfg.ReMatchCgroupV2 {
			cg := proc.ReadCGroup(pid)
			for _, ba := range cfg.BadnessAdjReCgroupV2 {
				if ba.Regexp.MatchString(cg.V2) {
					if ba.Adjustment <= 0 {
						badness += ba.Adjustment
					} else if readAdj() >= 0 {
						badness += ba.Adjustment
					}
				}
			}
		}

		if cfg.ReMatchRealpath {
			rp := proc.ReadExeRealpath(pid)
			for _, ba := range cfg.BadnessAdjReRealpath {
				if ba.Regexp.MatchString(rp) {
					if ba.Adjustment <= 0 {
						badness += ba.Adjustment
					} else if readAdj() >= 0 {
						badness += ba.Adjustment
					}
				}
			}
		}

		if cfg.ReMatchCwd {
			cwd := proc.ReadCwd(pid)
			for _, ba := range cfg.BadnessAdjReCwd {
				if ba.Regexp.MatchString(cwd) {
					if ba.Adjustment <= 0 {
						badness += ba.Adjustment
					} else if readAdj() >= 0 {
						badness += ba.Adjustment
					}
				}
			}
		}

		if cfg.ReMatchCmdline {
			cmdline := proc.ReadCmdline(pid)
			for _, ba := range cfg.BadnessAdjReCmdline {
				if ba.Regexp.MatchString(cmdline) {
					if ba.Adjustment <= 0 {
						badness += ba.Adjustment
					} else if readAdj() >= 0 {
						badness += ba.Adjustment
					}
				}
			}
		}

		if cfg.ReMatchEnviron {
			env := proc.ReadEnviron(pid)
			for _, ba := range cfg.BadnessAdjReEnviron {
				if ba.Regexp.MatchString(env) {
					if ba.Adjustment <= 0 {
						badness += ba.Adjustment
					} else if readAdj() >= 0 {
						badness += ba.Adjustment
					}
				}
			}
		}

		if cfg.ReMatchUID {
			ps, err := proc.ReadProcessStatus(pid)
			uidStr := ""
			if err == nil {
				uidStr = fmt.Sprintf("%d", ps.UID)
			}
			for _, ba := range cfg.BadnessAdjReUID {
				if ba.Regexp.MatchString(uidStr) {
					if ba.Adjustment <= 0 {
						badness += ba.Adjustment
					} else if readAdj() >= 0 {
						badness += ba.Adjustment
					}
				}
			}
		}
	}

	if badness < 0 {
		badness = 0
	}

	return badness, oomScore
}

func FindVictim(selfPID int, cfg *config.Config) *Result {
	pids, err := proc.PIDs()
	if err != nil {
		return nil
	}
	var best *Result
	for _, pid := range pids {
		if pid == 1 || pid == selfPID {
			continue
		}
		oom := proc.ReadOOMScore(pid)
		if oom < 1 {
			continue
		}
		if !proc.IsAlive(pid) {
			continue
		}
		badness, _ := Calculate(pid, oom, cfg)
		if badness == 0 {
			continue
		}
		if best == nil || badness > best.Badness {
			best = &Result{PID: pid, Badness: badness, OOMScore: oom}
		}
	}
	return best
}
