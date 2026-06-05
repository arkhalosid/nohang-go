package action

import (
	"context"
	"fmt"
	"github.com/user/nohang/internal/badness"
	"github.com/user/nohang/internal/config"
	"github.com/user/nohang/internal/notifier"
	"github.com/user/nohang/internal/proc"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type VictimCache struct {
	mu      sync.Mutex
	entries map[string]*CacheEntry
	ttl     float64
}

type CacheEntry struct {
	Time time.Time
	Name string
}

func NewVictimCache(ttl float64) *VictimCache {
	return &VictimCache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
	}
}

func (vc *VictimCache) Set(victimID, name string) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.entries[victimID] = &CacheEntry{Time: time.Now(), Name: name}
}

func (vc *VictimCache) Cleanup() {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	for id, e := range vc.entries {
		if time.Since(e.Time).Seconds() > vc.ttl {
			delete(vc.entries, id)
			continue
		}
		iva := proc.IsVictimAlive(id)
		if iva == 0 || iva == 3 {
			delete(vc.entries, id)
		}
	}
}

func (vc *VictimCache) FindCached(cacheTime float64) (victimID, name string, pid int, ok bool) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	for id, entry := range vc.entries {
		if time.Since(entry.Time).Seconds() < cacheTime {
			pid = parsePID(id)
			return id, entry.Name, pid, true
		}
	}
	return "", "", 0, false
}

func parsePID(victimID string) int {
	var pid int
	fmt.Sscanf(victimID, "_pid%d", &pid)
	return pid
}

func sigName(sig int) string {
	switch sig {
	case 9:
		return "SIGKILL"
	case 15:
		return "SIGTERM"
	case 1:
		return "SIGHUP"
	case 2:
		return "SIGINT"
	case 3:
		return "SIGQUIT"
	}
	return fmt.Sprintf("SIGUNKN(%d)", sig)
}

func ApplyCorrectiveAction(
	threshold int,
	cfg *config.Config,
	selfPID int,
	victimCache *VictimCache,
	lastActionTime *time.Time,
	notif *notifier.Notifier,
	logFn func(string, ...interface{}),
	updateStat func(string),
	printStat func(),
	sleepFn func(time.Duration),
	overSleep time.Duration,
) {
	logFn(">>=== STARTING implement_corrective_action() ====>>")

	if time.Since(*lastActionTime).Seconds() < 1.0 {
		logFn("Time since OOM: %.3fs; post OOM delay (1s) is not exceeded",
			time.Since(*lastActionTime).Seconds())
		logFn("<<=== FINISHING implement_corrective_action() ===<<")
		return
	}

	victimCache.Cleanup()
	time0 := time.Now()

	victimID, victimName, pid, found := victimCache.FindCached(cfg.VictimCacheTime)
	var victimBadness int

	if found {
		logFn("New victim is cached victim %d (%s)", pid, victimName)
	} else {
		result := badness.FindVictim(selfPID, cfg)
		if result == nil {
			logFn("Sleep %.1fs", overSleep.Seconds())
			sleepFn(overSleep)
			logFn("<<=== FINISHING implement_corrective_action() ===<<")
			return
		}
		pid = result.PID
		victimBadness = result.Badness
		victimName = proc.ReadComm(pid)
		victimID = proc.GetVictimID(pid)
		sleepFn(100 * time.Millisecond)
	}

	if time.Since(*lastActionTime).Seconds() < 1.0 {
		logFn("Sleep %.1fs", overSleep.Seconds())
		sleepFn(overSleep)
		logFn("<<=== FINISHING implement_corrective_action() ===<<")
		return
	}

	if victimBadness < cfg.MinBadness {
		logFn("victim (PID: %d, Name: %s) badness (%d) < min_badness (%d); nothing to do",
			pid, victimName, victimBadness, cfg.MinBadness)
		updateStat("victim badness < min_badness")
		printStat()
		logFn("<<=== FINISHING implement_corrective_action() ===<<")
		return
	}

	logFn("Implementing a corrective action: Sending %s to the victim", sigName(threshold))

	err := syscall.Kill(pid, syscall.Signal(threshold))
	if err != nil {
		logFn("Cannot send signal to PID %d: %v", pid, err)
		updateStat(fmt.Sprintf("Cannot send %s to %s", sigName(threshold), victimName))
		printStat()
	} else {
		updateStat(fmt.Sprintf("[ OK ] Sending %s to %s", sigName(threshold), victimName))
		victimCache.Set(victimID, victimName)
		responseTime := time.Since(time0)
		logFn("OK; total response time: %.0fms", responseTime.Seconds()*1000)
		printStat()
	}

	*lastActionTime = time.Now()
	killTimestamp := time.Now()

	for {
		sleepFn(10 * time.Millisecond)
		d := time.Since(killTimestamp)
		iva := proc.IsVictimAlive(victimID)

		if iva == 0 {
			logFn("The victim died in %.3fs", d.Seconds())
			victimCache.mu.Lock()
			delete(victimCache.entries, victimID)
			victimCache.mu.Unlock()
			break
		} else if iva == 1 {
			if d.Seconds() > overSleep.Seconds()/4+10 {
				logFn("The victim doesn't respond on corrective action in %.3fs", d.Seconds())
				break
			}
		} else if iva == 3 {
			logFn("The victim became a zombie in %.3fs", d.Seconds())
			victimCache.mu.Lock()
			delete(victimCache.entries, victimID)
			victimCache.mu.Unlock()
			sleepFn(time.Duration(cfg.PostZombieDelay * float64(time.Second)))
			break
		}
	}

	if cfg.PostActionGUINotifications {
		title := "System hang prevention"
		body := fmt.Sprintf("<b>%s</b> [%d] <b>%s</b>", sigName(threshold), pid, victimName)
		if cfg.HideCorrectiveActionType {
			body = "Corrective action applied"
		}
		notif.Send(title, body)
	}

	logFn("<<=== FINISHING implement_corrective_action() ===<<")
}

func ExecCommandFn(cmd string, timeout float64, logFn func(string, ...interface{})) {
	parts := splitCommand(cmd)
	if len(parts) == 0 {
		return
	}
	logFn("Executing command: %s with timeout %.0fs", cmd, timeout)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout*float64(time.Second)))
	defer cancel()

	c := exec.CommandContext(ctx, parts[0], parts[1:]...)
	if output, err := c.CombinedOutput(); err != nil {
		logFn("Command error: %v, output: %s", err, string(output))
	} else {
		logFn("Command completed: %s", string(output))
	}
}

func splitCommand(cmd string) []string {
	var result []string
	current := ""
	inQuote := false
	for _, c := range cmd {
		if c == '\'' || c == '"' {
			inQuote = !inQuote
			continue
		}
		if c == ' ' && !inQuote {
			if current != "" {
				result = append(result, current)
				current = ""
			}
			continue
		}
		current += string(c)
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
