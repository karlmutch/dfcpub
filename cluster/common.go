/*
 * Copyright (c) 2018, NVIDIA CORPORATION. All rights reserved.
 *
 */
// Package cluster provides common interfaces and local access to cluster-level metadata
package cluster

import (
	"github.com/NVIDIA/dfcpub/cmn"
	"github.com/NVIDIA/dfcpub/ios"
)

// globals
var (
	ifs    CommonInterfaces
	config CommonConfig
)

type (
	// interfaces that are commonly used by other modules and packages
	CommonInterfaces struct {
		Sowner     Sowner
		Bowner     Bowner
		Riostat    *ios.IostatRunner
		NameLocker NameLocker
		Rebalancer Rebalancer
		Throttler  Throttler
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

	// a slice of configuration commonly used by other modules and packages
	CommonConfig struct {
		Log      *cmn.LogConfig
		Periodic *cmn.Periodic
		LRU      *cmn.LRUConfig
		Xaction  *cmn.XactionConfig
	}
)
