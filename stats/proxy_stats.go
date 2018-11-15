/*
 * Copyright (c) 2018, NVIDIA CORPORATION. All rights reserved.
 */
package stats

import (
	"time"

	"github.com/NVIDIA/dfcpub/3rdparty/glog"
	"github.com/NVIDIA/dfcpub/cmn"
	"github.com/NVIDIA/dfcpub/stats/statsd"
	jsoniter "github.com/json-iterator/go"
)

type (
	proxyCoreStats struct {
		Tracker statsTracker
		// omitempty
		statsdC *statsd.Client
		logged  bool
	}
	ProxyRunner struct {
		statsrunner
		Core *proxyCoreStats `json:"core"`
	}
	ClusterStats struct {
		Proxy  *proxyCoreStats             `json:"proxy"`
		Target map[string]*TargetRunner `json:"target"`
	}
	ClusterStatsRaw struct {
		Proxy  *proxyCoreStats                `json:"proxy"`
		Target map[string]jsoniter.RawMessage `json:"target"`
	}
)

func (p *proxyCoreStats) initStatsTracker() {
	p.Tracker = statsTracker(map[string]*statsInstance{})
	p.Tracker.registerCommonStats()
}

func (p *proxyCoreStats) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(p.Tracker)
}

func (p *proxyCoreStats) UnmarshalJSON(b []byte) error {
	return jsoniter.Unmarshal(b, &p.Tracker)
}

//
// ProxyRunner
//
func (r *ProxyRunner) Run() error {
	return r.runcommon(r)
}
func (r *ProxyRunner) Init(statsPeriod *time.Duration) {
	r.statsPeriod = statsPeriod
	r.Core = &proxyCoreStats{}
	r.Core.initStatsTracker()
}

// statslogger interface impl
func (r *ProxyRunner) log() (runlru bool) {
	r.Lock()
	if r.Core.logged {
		r.Unlock()
		return
	}
	for _, v := range r.Core.Tracker {
		if v.kind == statsKindLatency && v.associatedVal > 0 {
			v.Value /= v.associatedVal
		}
	}
	b, err := jsoniter.Marshal(r.Core)

	// reset all the latency stats only
	for _, v := range r.Core.Tracker {
		if v.kind == statsKindLatency {
			v.Value = 0
			v.associatedVal = 0
		}
	}
	r.Unlock()

	if err == nil {
		glog.Infoln(string(b))
		r.Core.logged = true
	}
	return
}

func (r *ProxyRunner) doAdd(nv NamedVal64) {
	r.Lock()
	s := r.Core
	s.doAdd(nv.name, nv.val)
	r.Unlock()
}

func (s *proxyCoreStats) doAdd(name string, val int64) {
	if v, ok := s.Tracker[name]; !ok {
		cmn.Assert(false, "Invalid stats name "+name)
	} else if v.kind == statsKindLatency {
		s.Tracker[name].associatedVal++
		s.statsdC.Send(name,
			metric{statsd.Counter, "count", 1},
			metric{statsd.Timer, "latency", float64(time.Duration(val) / time.Millisecond)})
		val = int64(time.Duration(val) / time.Microsecond)
	} else {
		switch name {
		case statPostCount, statDeleteCount, statRenameCount:
			s.statsdC.Send(name, metric{statsd.Counter, "count", val})
		}
	}
	s.Tracker[name].Value += val
	s.logged = false
}
