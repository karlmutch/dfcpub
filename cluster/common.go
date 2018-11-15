/*
 * Copyright (c) 2018, NVIDIA CORPORATION. All rights reserved.
 *
 */
// Package cluster provides common interfaces and local access to cluster-level metadata
package cluster

import (
	"github.com/NVIDIA/dfcpub/cmn"
)

// runners
const (
	Xproxy           = "proXy"
	Xtarget          = "target"
	Xmem             = "gmem2"
	Xsignal          = "signal"
	Xproxystats      = "proXystats"
	Xstorstats       = "storstats"
	Xproxykeepalive  = "proXykeepalive"
	Xtargetkeepalive = "targetkeepalive"
	Xiostat          = "iostat"
	Xatime           = "atime"
	Xmetasyncer      = "metasyncer"
	Xfshc            = "fshc"
	Xreadahead       = "readahead"
	Xreplication     = "replication"
)

// globals
var (
	// part of the configuration accessible by external modules and packages
	Config CommonConfig

	// runmap for external modules and packages to locate each other
	RunMap map[string]cmn.Runner
)

type (
	CommonConfig struct {
		Log      *cmn.LogConfig
		Periodic *cmn.Periodic
		LRU      *cmn.LRUConfig
		Xaction  *cmn.XactionConfig
	}
	// NameLocker interface locks and unlocks (and try-locks, etc.)
	// arbitrary strings.
	// NameLocker is currently utilized to lock objects stored in the cluster
	// when there's a pending GET, PUT, etc. transaction and we don't want
	// the object in question to get updated or evicted concurrently.
	// Objects are locked by their unique (string) names aka unames.
	// The lock can be exclusive (write) or shared (read).
	// For implementation, please refer to dfc/rtnames.go
	NameLocker interface {
		TryLock(uname string, exclusive bool) bool
		Lock(uname string, exclusive bool)
		DowngradeLock(uname string)
		Unlock(uname string, exclusive bool)
	}
	// For implementation, please refer to dfc/target.go
	Rebalancer interface {
		IsRebalancing() bool
	}
)

func GetProxyStatsRunner() cmn.Runner  { return RunMap[Xproxystats] }
func GetProxyKeepalive() cmn.Runner    { return RunMap[Xproxykeepalive] }
func GetTarget() cmn.Runner            { return RunMap[Xtarget] }
func GetMem2() cmn.Runner              { return RunMap[Xmem] }
func GetTargetKeepalive() cmn.Runner   { return RunMap[Xtargetkeepalive] }
func GetReplicationRunner() cmn.Runner { return RunMap[Xreplication] }
func GetTargetStatsRunner() cmn.Runner { return RunMap[Xstorstats] }
func GetIostatRunner() cmn.Runner      { return RunMap[Xiostat] }
func GetAtimeRunner() cmn.Runner       { return RunMap[Xatime] }
func GetCloudIf() cmn.Runner           { return RunMap[Xtarget] }
func GetMetasyncer() cmn.Runner        { return RunMap[Xmetasyncer] }
func GetFSHC() cmn.Runner              { return RunMap[Xfshc] }
