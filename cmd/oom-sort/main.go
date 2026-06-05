package main

import (
	"flag"
	"fmt"
	"github.com/user/nohang/internal/proc"
	"os"
	"sort"
	"strconv"
)

type ProcEntry struct {
	PID          int
	OOMScore     int
	OOMScoreAdj  int
	Cmdline      string
	Name         string
	UID          int
	VmRSS        float64
	VmSwap       float64
}

func main() {
	numLines := flag.Int("n", 99999, "max number of lines")
	cmdlineLen := flag.Int("l", 99999, "max cmdline length")
	sortBy := flag.String("s", "oom_score", "sort by unit")
	flag.Parse()

	sortKeys := map[string]int{
		"PID": 0, "oom_score": 1, "oom_score_adj": 2,
		"cmdline": 3, "Name": 4, "UID": 5, "VmRSS": 6, "VmSwap": 7,
	}

	sortKey, ok := sortKeys[*sortBy]
	if !ok {
		fmt.Fprintf(os.Stderr, "Invalid -s value. Valid: PID, oom_score, oom_score_adj, UID, Name, cmdline, VmRSS, VmSwap\n")
		os.Exit(1)
	}

	var entries []ProcEntry
	pids, err := proc.PIDs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, pid := range pids {
		if pid == 1 {
			continue
		}
		oomScore := proc.ReadOOMScore(pid)
		oomScoreAdj := proc.ReadOOMScoreAdj(pid)
		cmdline := proc.ReadCmdline(pid)
		if cmdline == "" {
			continue
		}
		ps, err := proc.ReadProcessStatus(pid)
		if err != nil {
			continue
		}
		entries = append(entries, ProcEntry{
			PID: pid, OOMScore: oomScore, OOMScoreAdj: oomScoreAdj,
			Cmdline: cmdline, Name: ps.Name, UID: ps.UID,
			VmRSS: ps.VmRSS, VmSwap: ps.VmSwap,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		vals := []int{entries[i].PID, entries[i].OOMScore, entries[i].OOMScoreAdj, 0, 0, entries[i].UID, int(entries[i].VmRSS), int(entries[i].VmSwap)}
		valj := []int{entries[j].PID, entries[j].OOMScore, entries[j].OOMScoreAdj, 0, 0, entries[j].UID, int(entries[j].VmRSS), int(entries[j].VmSwap)}
		return vals[sortKey] > valj[sortKey]
	})

	if *cmdlineLen == 0 {
		fmt.Printf("oom_score oom_score_adj UID  PID Name        VmRSS   VmSwap\n")
		fmt.Printf("--------- ------------- ---  --- ----------- ------  ------\n")
	} else {
		fmt.Printf("oom_score oom_score_adj UID  PID Name        VmRSS   VmSwap   cmdline\n")
		fmt.Printf("--------- ------------- ---  --- ----------- ------  ------   -------\n")
	}

	limit := *numLines
	if limit > len(entries) {
		limit = len(entries)
	}

	for _, e := range entries[:limit] {
		cl := e.Cmdline
		if len(cl) > *cmdlineLen {
			cl = cl[:*cmdlineLen]
		}
		if *cmdlineLen == 0 {
			fmt.Printf("%9d %13d %3d %3d %-15s %5d M %6d M\n",
				e.OOMScore, e.OOMScoreAdj, e.UID, e.PID, e.Name,
				int(e.VmRSS/1024), int(e.VmSwap/1024))
		} else {
			fmt.Printf("%9d %13d %3d %3d %-15s %5d M %6d M %s\n",
				e.OOMScore, e.OOMScoreAdj, e.UID, e.PID, e.Name,
				int(e.VmRSS/1024), int(e.VmSwap/1024), cl)
		}
	}
}

func strToInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func strToFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
