package proc

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func PIDs() ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var pids []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

func IsAlive(pid int) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err != nil {
		return false
	}
	parts := strings.Fields(string(data))
	if len(parts) < 2 {
		return false
	}
	rss := parts[1]
	return rss != "0"
}

func AlivePIDs(selfPID int) ([]int, error) {
	pids, err := PIDs()
	if err != nil {
		return nil, err
	}
	var alive []int
	for _, pid := range pids {
		if pid == 1 || pid == selfPID {
			continue
		}
		if IsAlive(pid) {
			alive = append(alive, pid)
		}
	}
	return alive, nil
}

type ProcessStatus struct {
	Name     string
	State    string
	PPID     int
	UID      int
	VmSize   float64
	VmRSS    float64
	VmSwap   float64
	RssAnon  float64
	RssFile  float64
	RssShmem float64
}

var cachedStatusIndexes map[string]int

func getStatusIndexes() map[string]int {
	if cachedStatusIndexes != nil {
		return cachedStatusIndexes
	}
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return nil
	}
	cachedStatusIndexes = make(map[string]int)
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			cachedStatusIndexes[parts[0]] = i
		}
	}
	return cachedStatusIndexes
}

func ReadProcessStatus(pid int) (*ProcessStatus, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return nil, err
	}
	ps := &ProcessStatus{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch name {
		case "Name":
			ps.Name = val
		case "State":
			if len(val) > 0 {
				ps.State = string(val[0])
			}
		case "PPid":
			ps.PPID, _ = strconv.Atoi(val)
		case "Uid":
			uidParts := strings.Fields(val)
			if len(uidParts) > 0 {
				ps.UID, _ = strconv.Atoi(uidParts[0])
			}
		case "VmSize":
			val = strings.TrimSuffix(val, " kB")
			v, _ := strconv.ParseFloat(val, 64)
			ps.VmSize = v
		case "VmRSS":
			val = strings.TrimSuffix(val, " kB")
			v, _ := strconv.ParseFloat(val, 64)
			ps.VmRSS = v
		case "VmSwap":
			val = strings.TrimSuffix(val, " kB")
			v, _ := strconv.ParseFloat(val, 64)
			ps.VmSwap = v
		case "RssAnon":
			val = strings.TrimSuffix(val, " kB")
			v, _ := strconv.ParseFloat(val, 64)
			ps.RssAnon = v
		case "RssFile":
			val = strings.TrimSuffix(val, " kB")
			v, _ := strconv.ParseFloat(val, 64)
			ps.RssFile = v
		case "RssShmem":
			val = strings.TrimSuffix(val, " kB")
			v, _ := strconv.ParseFloat(val, 64)
			ps.RssShmem = v
		}
	}
	return ps, nil
}

func ReadComm(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(data), "\n")
}

func ReadCmdline(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(strings.TrimRight(string(data), "\x00"), "\x00", " ")
}

func ReadEnviron(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(strings.TrimRight(string(data), "\x00"), "\x00", " ")
}

func ReadExeRealpath(pid int) string {
	path, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return ""
	}
	return path
}

func ReadCwd(pid int) string {
	path, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	if err != nil {
		return ""
	}
	return path
}

type CGroupInfo struct {
	V1 string
	V2 string
}

func GetCGroupV1Index() int {
	return findCGroupIndex(":")
}

func GetCGroupV2Index() int {
	return findCGroupIndex("0::")
}

var cgroupV1Index, cgroupV2Index int

func init() {
	cgroupV1Index, cgroupV2Index = findCGroupIndexes()
}

func findCGroupIndexes() (int, int) {
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return -1, -1
	}
	v1idx, v2idx := -1, -1
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.Contains(line, ":name=") {
			v1idx = i
		}
		if strings.HasPrefix(line, "0::") {
			v2idx = i
		}
	}
	return v1idx, v2idx
}

func ReadCGroup(pid int) *CGroupInfo {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return &CGroupInfo{}
	}
	ci := &CGroupInfo{}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if i == cgroupV1Index && cgroupV1Index >= 0 {
			idx := strings.Index(line, ":")
			if idx >= 0 {
				after := line[idx+1:]
				idx2 := strings.Index(after, ":")
				if idx2 >= 0 {
					ci.V1 = "/" + after[idx2+1:]
				}
			}
		}
		if i == cgroupV2Index && cgroupV2Index >= 0 {
			if strings.HasPrefix(line, "0::") {
				ci.V2 = line[3:]
			}
		}
	}
	return ci
}

func findCGroupIndex(prefix string) int {
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return -1
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return i
		}
	}
	return -1
}

func ReadStarttime(pid int) float64 {
	stat, err := readStat(pid)
	if err != nil {
		return 0
	}
	return stat.Starttime
}

func ReadNSSID(pid int) int {
	stat, err := readStat(pid)
	if err != nil {
		return 0
	}
	return stat.NSSID
}

type ProcStat struct {
	Comm      string
	State     string
	PPID      int
	Session   int
	Starttime float64
	NSSID     int
}

func readStat(pid int) (*ProcStat, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return nil, err
	}
	s := string(data)
	right := strings.LastIndex(s, ")")
	if right < 0 {
		return nil, fmt.Errorf("bad stat format")
	}
	fields := strings.Fields(s[right+1:])
	ps := &ProcStat{}
	if len(fields) > 0 {
		ps.State = fields[0]
	}
	if len(fields) > 2 {
		ps.PPID, _ = strconv.Atoi(fields[2])
	}
	if len(fields) > 3 {
		ps.Session, _ = strconv.Atoi(fields[3])
	}
	if len(fields) > 19 {
		ps.Starttime, _ = strconv.ParseFloat(fields[19], 64)
	}
	ps.Comm = s[:right]
	return ps, nil
}

func GetVictimID(pid int) string {
	starttime := ReadStarttime(pid)
	return fmt.Sprintf("%.0f_pid%d", starttime, pid)
}

func IsVictimAlive(victimID string) int {
	parts := strings.Split(victimID, "_pid")
	if len(parts) != 2 {
		return 0
	}
	pid, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	newID := GetVictimID(pid)
	if victimID != newID {
		return 0
	}
	if IsAlive(pid) {
		return 1
	}
	ps, err := ReadProcessStatus(pid)
	if err != nil {
		return 0
	}
	switch ps.State {
	case "R":
		return 2
	case "Z":
		return 3
	case "X", "":
		return 0
	}
	return 0
}

func ReadAncestry(pid int, depth int) string {
	if depth == 0 {
		return ""
	}
	var parts []string
	current := pid
	for i := 0; i < depth; i++ {
		ps, err := ReadProcessStatus(current)
		if err != nil || ps.PPID == 0 {
			break
		}
		pname := ReadComm(ps.PPID)
		parts = append(parts, fmt.Sprintf("PID %d (%s)", ps.PPID, pname))
		if ps.PPID == 1 {
			break
		}
		current = ps.PPID
	}
	if len(parts) == 0 {
		return ""
	}
	return "\n  ancestry:  " + strings.Join(parts, " <= ")
}

func GetSCPageSize() int64 {
	return 4096
}

func GetSCClkTck() float64 {
	return 100.0
}

func readProcFile(path string) string {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(data), "\n")
}

func ReadOOMScore(pid int) int {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/oom_score", pid))
	if err != nil {
		return 0
	}
	v, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return v
}

func ReadOOMScoreAdj(pid int) int {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/oom_score_adj", pid))
	if err != nil {
		return 0
	}
	v, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return v
}
