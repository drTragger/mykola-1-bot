package utils

import "sync"

type sysMetrics struct {
	cpuUsage, cpuFreq                                           string
	cpuCount                                                    int
	ramUsage, zramUsage, swapUsage                              string
	diskUsage, diskFree, temperature, loadAvg, uptime, bootTime string
	procCount, usersCount                                       int
	netRx, netTx, netRxSpeed, netTxSpeed                        string
	publicIP, ping, throttled, servicesBlock                    string

	mu sync.Mutex
}
