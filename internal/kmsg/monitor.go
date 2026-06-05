package kmsg

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Monitor struct {
	mu              sync.RWMutex
	lastOOMTime     time.Time
	lastActionTime  time.Time
	calibString     string
	calibMono       float64
	kmsgTimeDelta   float64
	calibrated      bool
	debug           bool
	log             func(format string, args ...interface{})
	updateStat      func(key string)
	printStat       func()
	guiNotify       func(title, body string)
	postActionGUINotifications bool
}

func New(debug bool, logFn func(string, ...interface{}), updateStat func(string), printStat func()) *Monitor {
	return &Monitor{
		debug:      debug,
		log:        logFn,
		updateStat: updateStat,
		printStat:  printStat,
	}
}

func (m *Monitor) SetGUINotify(fn func(title, body string)) {
	m.postActionGUINotifications = true
	m.guiNotify = fn
}

func (m *Monitor) IsKMsgOK() bool {
	m.calibString = fmt.Sprintf("nohang: clock calibration: %d\n", time.Now().UnixNano())
	if err := os.WriteFile("/dev/kmsg", []byte(m.calibString), 0644); err != nil {
		m.log("kmsg write test failed: %v", err)
		return false
	}
	return true
}

func (m *Monitor) GetLastOOMTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastOOMTime
}

func (m *Monitor) GetLastActionTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastActionTime
}

func (m *Monitor) Run() {
	f, err := os.Open("/dev/kmsg")
	if err != nil {
		m.log("kmsg: cannot open: %v", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	startTime := time.Now()
	var linesRead int
	calibrated := false

	for scanner.Scan() {
		line := scanner.Text()

		if !calibrated {
			if strings.Contains(line, m.calibString) {
				parts := strings.Split(line, ",")
				if len(parts) >= 3 {
					kmsgMono, _ := strconv.ParseFloat(parts[2], 64)
					kmsgMono /= 1e6
					m.kmsgTimeDelta = kmsgMono - float64(time.Now().UnixNano())/1e9
					calibrated = true
					m.log("Checking kmsg for OOM events has started in %.0fs; %d lines parsed",
						time.Since(startTime).Seconds(), linesRead)
				}
			}
			linesRead++
			if time.Since(startTime) > 10*time.Second {
				m.log("kmsg: cannot start in 10s")
				return
			}
			continue
		}

		m.processLine(line)
	}
}

func (m *Monitor) processLine(s string) {
	parts := strings.Split(s, ",")
	if len(parts) < 3 {
		return
	}

	kmsgMono, _ := strconv.ParseFloat(parts[2], 64)
	kmsgMono /= 1e6
	realTime := time.Unix(0, int64((kmsgMono-m.kmsgTimeDelta)*1e9))

	switch {
	case strings.HasPrefix(s, "3,") && strings.Contains(s, "ut of memory: Kill"):
		if m.debug {
			m.log("debug kmsg: %s", s)
		}
		m.mu.Lock()
		m.lastOOMTime = realTime
		m.lastActionTime = realTime
		m.mu.Unlock()
		msg := s[strings.LastIndex(s, ";")+1:]
		m.log("kmsg: %s", msg)
		m.updateStat("kmsg: Out of memory: Kill")

	case strings.HasPrefix(s, "6,"):
		if strings.Contains(s, "oom_reaper: reaped process ") {
			if m.debug {
				m.log("debug kmsg: %s", s)
			}
			m.mu.Lock()
			m.lastActionTime = realTime
			m.mu.Unlock()
			msg := s[strings.LastIndex(s, ";")+1:]
			m.log("kmsg: %s", msg)
			m.updateStat("kmsg: oom_reaper: reaped process")
		}
		if strings.Contains(s, "killed due to memory.oom.group set") {
			if m.debug {
				m.log("debug kmsg: %s", s)
			}
			m.mu.Lock()
			m.lastOOMTime = realTime
			m.lastActionTime = realTime
			m.mu.Unlock()
			msg := s[strings.LastIndex(s, ";")+1:]
			m.log("kmsg: %s", msg)
			m.updateStat("killed due to memory.oom.group set")
		}

	case strings.HasPrefix(s, "4,") && strings.Contains(s, " invoked oom-killer: "):
		if m.debug {
			m.log("debug kmsg: %s", s)
		}
		m.mu.Lock()
		m.lastOOMTime = realTime
		m.lastActionTime = realTime
		m.mu.Unlock()
		msg := s[strings.LastIndex(s, ";")+1:]
		m.log("kmsg: %s", msg)
		m.updateStat("kmsg: invoked oom-killer")
		m.printStat()
		if m.postActionGUINotifications && m.guiNotify != nil {
			m.guiNotify("kmsg: Out of memory!", "invoked oom-killer")
		}
	}
}
