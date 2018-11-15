/*
 * Copyright (c) 2018, NVIDIA CORPORATION. All rights reserved.
 */
package stats

import (
	"net/http"
	"sync"
	"time"

	"github.com/NVIDIA/dfcpub/3rdparty/glog"
	"github.com/NVIDIA/dfcpub/cmn"
	"github.com/NVIDIA/dfcpub/stats/statsd"
	jsoniter "github.com/json-iterator/go"
)

const (
	statsKindCounter = "counter"
	statsKindLatency = "latency"
)

const logsTotalSizeCheckTime = time.Hour * 3

// Stats common to proxyCoreStats and targetCoreStats
const (
	statGetCount            = "get.n"
	statPutCount            = "put.n"
	statPostCount           = "pst.n"
	statDeleteCount         = "del.n"
	statRenameCount         = "ren.n"
	statListCount           = "lst.n"
	statGetLatency          = "get.μs"
	statListLatency         = "lst.μs"
	statKeepAliveMinLatency = "kalive.μs.min"
	statKeepAliveMaxLatency = "kalive.μs.max"
	statKeepAliveLatency    = "kalive.μs"
	statUptimeLatency       = "uptime.μs"
	statErrCount            = "err.n"
	statErrGetCount         = "err.get.n"
	statErrDeleteCount      = "err.delete.n"
	statErrPostCount        = "err.post.n"
	statErrPutCount         = "err.put.n"
	statErrHeadCount        = "err.head.n"
	statErrListCount        = "err.list.n"
	statErrRangeCount       = "err.range.n"
)

type (
	metric = statsd.Metric // type alias
	// is implemented by the stats runners
	statslogger interface {
		log() (runlru bool)
		housekeep(bool)
		doAdd(nv NamedVal64)
	}
	// is implemented by the *CoreStats types
	Tracker interface {
		add(name string, val int64)
		addErrorHTTP(method string, val int64)
		addMany(NamedVal64 ...NamedVal64)
	}
	NamedVal64 struct {
		name string
		val  int64
	}
	statsrunner struct {
		sync.RWMutex
		cmn.Named
		statsPeriod *time.Duration
		stopCh      chan struct{}
		workCh      chan NamedVal64
		starttime   time.Time
	}
	// Stats are tracked via a map of stats names (key) to statInstances (values).
	// There are two main types of stats: counter and latency declared
	// using the the kind field. Only latency stats have associatedVals to them
	// that are used in calculating latency measurements.
	statsInstance struct {
		Value         int64 `json:"value"`
		kind          string
		associatedVal int64
	}
	statsTracker map[string]*statsInstance
)

func (stats statsTracker) register(key string, kind string) {
	cmn.Assert(kind == statsKindCounter || kind == statsKindLatency, "Invalid stats kind "+kind)
	stats[key] = &statsInstance{0, kind, 0}
}

// These stats are common to proxyCoreStats and targetCoreStats
func (stats statsTracker) registerCommonStats() {
	cmn.Assert(stats != nil, "Error attempting to register stats into nil map")

	stats.register(statGetCount, statsKindCounter)
	stats.register(statPutCount, statsKindCounter)
	stats.register(statPostCount, statsKindCounter)
	stats.register(statDeleteCount, statsKindCounter)
	stats.register(statRenameCount, statsKindCounter)
	stats.register(statListCount, statsKindCounter)
	stats.register(statGetLatency, statsKindCounter)
	stats.register(statListLatency, statsKindLatency)
	stats.register(statKeepAliveMinLatency, statsKindLatency)
	stats.register(statKeepAliveMaxLatency, statsKindLatency)
	stats.register(statKeepAliveLatency, statsKindLatency)
	stats.register(statUptimeLatency, statsKindLatency)
	stats.register(statErrCount, statsKindCounter)
	stats.register(statErrGetCount, statsKindCounter)
	stats.register(statErrDeleteCount, statsKindCounter)
	stats.register(statErrPostCount, statsKindCounter)
	stats.register(statErrPutCount, statsKindCounter)
	stats.register(statErrHeadCount, statsKindCounter)
	stats.register(statErrListCount, statsKindCounter)
	stats.register(statErrRangeCount, statsKindCounter)
}

func (stat *statsInstance) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(stat.Value)
}

func (stat *statsInstance) UnmarshalJSON(b []byte) error {
	return jsoniter.Unmarshal(b, &stat.Value)
}

//
// statsrunner
//

// implements Tracker interface
var _ Tracker = &statsrunner{}

func (r *statsrunner) runcommon(logger statslogger) error {
	r.stopCh = make(chan struct{}, 4)
	r.workCh = make(chan NamedVal64, 256)
	r.starttime = time.Now()

	glog.Infof("Starting %s", r.Getname())
	ticker := time.NewTicker(*r.statsPeriod)
	for {
		select {
		case nv, ok := <-r.workCh:
			if ok {
				logger.doAdd(nv)
			}
		case <-ticker.C:
			runlru := logger.log()
			logger.housekeep(runlru)
		case <-r.stopCh:
			ticker.Stop()
			return nil
		}
	}
}

func (r *statsrunner) Stop(err error) {
	glog.Infof("Stopping %s, err: %v", r.Getname(), err)
	r.stopCh <- struct{}{}
	close(r.stopCh)
}

// statslogger interface impl
func (r *statsrunner) log() (runlru bool)  { return false }
func (r *statsrunner) housekeep(bool)      {}
func (r *statsrunner) doAdd(nv NamedVal64) {}

func (r *statsrunner) addMany(nvs ...NamedVal64) {
	for _, nv := range nvs {
		r.workCh <- nv
	}
}

func (r *statsrunner) add(name string, val int64) {
	r.workCh <- NamedVal64{name, val}
}

func (r *statsrunner) addErrorHTTP(method string, val int64) {
	switch method {
	case http.MethodGet:
		r.workCh <- NamedVal64{statErrGetCount, val}
	case http.MethodDelete:
		r.workCh <- NamedVal64{statErrDeleteCount, val}
	case http.MethodPost:
		r.workCh <- NamedVal64{statErrPostCount, val}
	case http.MethodPut:
		r.workCh <- NamedVal64{statErrPutCount, val}
	case http.MethodHead:
		r.workCh <- NamedVal64{statErrHeadCount, val}
	default:
		r.workCh <- NamedVal64{statErrCount, val}
	}
}
