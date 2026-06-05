package notifier

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Notifier struct {
	mu          sync.Mutex
	envCache    []EnvEntry
	envCacheTime time.Time
	cacheTTL    float64
	debug       bool
	log         func(format string, args ...interface{})
}

type EnvEntry struct {
	User   string
	Display string
	DBus   string
}

func New(cacheTTL float64, debug bool, logFn func(string, ...interface{})) *Notifier {
	return &Notifier{
		cacheTTL: cacheTTL,
		debug:    debug,
		log:      logFn,
	}
}

func (n *Notifier) Send(title, body string) {
	uid := os.Geteuid()
	args := []string{"notify-send", "--icon=dialog-warning", title, body}

	if uid != 0 {
		cmd := exec.Command(args[0], args[1:]...)
		go func() {
			if err := cmd.Run(); err != nil && n.debug {
				n.log("notify-send error: %v", err)
			}
		}()
		return
	}

	envs := n.getEnvList()
	if len(envs) == 0 {
		if n.debug {
			n.log("Nobody logged-in with GUI. Nothing to do.")
		}
		return
	}

	for _, env := range envs {
		cmd := exec.Command("sudo", "-u", env.User, "env",
			fmt.Sprintf("DISPLAY=%s", env.Display),
			fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=%s", env.DBus),
			"notify-send", "--icon=dialog-warning", "--app-name=nohang",
			title, body)
		go func() {
			if err := cmd.Run(); err != nil && n.debug {
				n.log("notify-send error: %v", err)
			}
		}()
	}
}

func (n *Notifier) getEnvList() []EnvEntry {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.envCache != nil && time.Since(n.envCacheTime).Seconds() < n.cacheTTL {
		return n.envCache
	}

	var envs []EnvEntry
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid := e.Name()
		if pid[0] < '0' || pid[0] > '9' {
			continue
		}
		env := readProcessEnviron(pid)
		if env != nil {
			envs = append(envs, *env)
		}
	}

	uniq := make(map[string]EnvEntry)
	for _, e := range envs {
		key := e.User + ":" + e.Display
		uniq[key] = e
	}
	var result []EnvEntry
	for _, e := range uniq {
		result = append(result, e)
	}

	n.envCache = result
	n.envCacheTime = time.Now()
	return result
}

func readProcessEnviron(pid string) *EnvEntry {
	data, err := os.ReadFile("/proc/" + pid + "/environ")
	if err != nil {
		return nil
	}

	var user, display, dbus string
	parts := strings.Split(string(data), "\x00")
	for _, p := range parts {
		if strings.HasPrefix(p, "HOME=/var") {
			return nil
		}
		if strings.HasPrefix(p, "USER=") {
			u := p[5:]
			if u == "root" {
				return nil
			}
			user = u
		} else if strings.HasPrefix(p, "DISPLAY=") {
			d := p[8:]
			if len(d) > 2 && d[len(d)-2] == '.' {
				d = d[:len(d)-2]
			}
			if len(d) > 10 {
				return nil
			}
			display = d
		} else if strings.HasPrefix(p, "DBUS_SESSION_BUS_ADDRESS=") {
			dbus = p[25:]
			if idx := strings.Index(dbus, ",guid="); idx >= 0 {
				dbus = dbus[:idx]
			}
		}
	}

	if user != "" && display != "" && dbus != "" {
		return &EnvEntry{User: user, Display: display, DBus: dbus}
	}
	return nil
}
