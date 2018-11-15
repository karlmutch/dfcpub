/*
 * Copyright (c) 2018, NVIDIA CORPORATION. All rights reserved.
 */
package stats

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/NVIDIA/dfcpub/3rdparty/glog"
	"github.com/NVIDIA/dfcpub/cmn"
	"github.com/NVIDIA/dfcpub/fs"
	"github.com/NVIDIA/dfcpub/stats/statsd"
	jsoniter "github.com/json-iterator/go"
)

// Stats only found in target
const (
	statPutLatency       = "put.μs"
	statGetColdCount     = "get.cold.n"
	statGetColdSize      = "get.cold.size"
	statLruEvictSize     = "lru.evict.size"
	statLruEvictCount    = "lru.evict.n"
	statTxCount          = "tx.n"
	statTxSize           = "tx.size"
	statRxCount          = "rx.n"
	statRxSize           = "rx.size"
	statPrefetchCount    = "pre.n"
	statPrefetchSize     = "pre.size"
	statVerChangeCount   = "vchange.n"
	statVerChangeSize    = "vchange.size"
	statErrCksumCount    = "err.cksum.n"
	statErrCksumSize     = "err.cksum.size"
	statGetRedirLatency  = "get.redir.μs"
	statPutRedirLatency  = "put.redir.μs"
	statRebalGlobalCount = "reb.global.n"
	statRebalLocalCount  = "reb.local.n"
	statRebalGlobalSize  = "reb.global.size"
	statRebalLocalSize   = "reb.local.size"
	statReplPutCount     = "replication.put.n"
	statReplPutLatency   = "replication.put.µs"
)

type (
	fscapacity struct {
		Used    uint64 `json:"used"`    // bytes
		Avail   uint64 `json:"avail"`   // ditto
		Usedpct int64  `json:"usedpct"` // reduntant ok
	}
	targetCoreStats struct {
		proxyCoreStats
	}
	storstatsrunner struct {
		statsrunner
		// init
		capUpdPeriod *time.Duration
		logMaxTotal  *uint64
		logDir       string
		lruHighWM    *int64
		lruEnabled   *bool
		// runtime
		Core     *targetCoreStats       `json:"core"`
		Capacity map[string]*fscapacity `json:"capacity"`
		// iostat
		CPUidle string                   `json:"cpuidle"`
		Disk    map[string]cmn.SimpleKVs `json:"disk"`
		// omitempty
		timeUpdatedCapacity time.Time
		timeCheckedLogSizes time.Time
		fsmap               map[syscall.Fsid]string
	}
)

//
// targetCoreStats
//

func (t *targetCoreStats) initStatsTracker() {
	// Call the embedded procxyCoreStats init method then register our own stats
	t.proxyCoreStats.initStatsTracker()

	t.Tracker.register(statPutLatency, statsKindLatency)
	t.Tracker.register(statGetColdCount, statsKindCounter)
	t.Tracker.register(statGetColdSize, statsKindCounter)
	t.Tracker.register(statLruEvictSize, statsKindCounter)
	t.Tracker.register(statLruEvictCount, statsKindCounter)
	t.Tracker.register(statTxCount, statsKindCounter)
	t.Tracker.register(statTxSize, statsKindCounter)
	t.Tracker.register(statRxCount, statsKindCounter)
	t.Tracker.register(statRxSize, statsKindCounter)
	t.Tracker.register(statPrefetchCount, statsKindCounter)
	t.Tracker.register(statPrefetchSize, statsKindCounter)
	t.Tracker.register(statVerChangeCount, statsKindCounter)
	t.Tracker.register(statVerChangeSize, statsKindCounter)
	t.Tracker.register(statErrCksumCount, statsKindCounter)
	t.Tracker.register(statErrCksumSize, statsKindCounter)
	t.Tracker.register(statGetRedirLatency, statsKindLatency)
	t.Tracker.register(statPutRedirLatency, statsKindLatency)
	t.Tracker.register(statRebalGlobalCount, statsKindCounter)
	t.Tracker.register(statRebalLocalCount, statsKindCounter)
	t.Tracker.register(statRebalGlobalSize, statsKindCounter)
	t.Tracker.register(statRebalLocalSize, statsKindCounter)
	t.Tracker.register(statReplPutCount, statsKindCounter)
	t.Tracker.register(statReplPutLatency, statsKindLatency)
}

func (t *targetCoreStats) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(t.Tracker)
}

func (t *targetCoreStats) UnmarshalJSON(b []byte) error {
	return jsoniter.Unmarshal(b, &t.Tracker)
}

func (s *targetCoreStats) doAdd(name string, val int64) {
	if _, ok := s.Tracker[name]; !ok {
		cmn.Assert(false, "Invalid stats name "+name)
	}

	switch name {
	// common
	case statGetCount, statPutCount, statPostCount, statDeleteCount, statRenameCount, statListCount,
		statGetLatency, statPutLatency, statListLatency,
		statKeepAliveLatency, statKeepAliveMinLatency, statKeepAliveMaxLatency,
		statErrCount, statErrGetCount, statErrDeleteCount, statErrPostCount,
		statErrPutCount, statErrHeadCount, statErrListCount, statErrRangeCount:
		s.proxyCoreStats.doAdd(name, val)
		return
	// target only
	case statGetColdSize:
		s.statsdC.Send("get.cold",
			metric{statsd.Counter, "count", 1},
			metric{statsd.Counter, "get.cold.size", val})
	case statVerChangeSize:
		s.statsdC.Send("get.cold",
			metric{statsd.Counter, "vchanged", 1},
			metric{statsd.Counter, "vchange.size", val})
	case statLruEvictSize, statTxSize, statRxSize, statErrCksumSize: // byte stats
		s.statsdC.Send(name, metric{statsd.Counter, "bytes", val})
	case statLruEvictCount, statTxCount, statRxCount: // files stats
		s.statsdC.Send(name, metric{statsd.Counter, "files", val})
	case statErrCksumCount: // counter stats
		s.statsdC.Send(name, metric{statsd.Counter, "count", val})
	case statGetRedirLatency, statPutRedirLatency: // latency stats
		s.Tracker[name].associatedVal++
		s.statsdC.Send(name,
			metric{statsd.Counter, "count", 1},
			metric{statsd.Timer, "latency", float64(time.Duration(val) / time.Millisecond)})
		val = int64(time.Duration(val) / time.Microsecond)
	}
	s.Tracker[name].Value += val
	s.logged = false
}

//
// storstatsrunner
//

func newFSCapacity(statfs *syscall.Statfs_t) *fscapacity {
	pct := (statfs.Blocks - statfs.Bavail) * 100 / statfs.Blocks
	return &fscapacity{
		Used:    (statfs.Blocks - statfs.Bavail) * uint64(statfs.Bsize),
		Avail:   statfs.Bavail * uint64(statfs.Bsize),
		Usedpct: int64(pct),
	}
}

func (r *storstatsrunner) Run() error {
	return r.runcommon(r)
}

func (r *storstatsrunner) Init(statsPeriod, capUpdPeriod *time.Duration,
	logMaxTotal *uint64, logDir string, lruHighWM *int64, lruEnabled *bool) {
	r.statsPeriod = statsPeriod
	r.capUpdPeriod = capUpdPeriod
	r.logMaxTotal = logMaxTotal
	r.logDir = logDir
	r.lruHighWM = lruHighWM
	r.lruEnabled = lruEnabled
	r.Disk = make(map[string]cmn.SimpleKVs, 8)
	r.updateCapacity()
	r.Core = &targetCoreStats{}
	r.Core.initStatsTracker()
}

func (r *storstatsrunner) log() (runlru bool) {
	r.Lock()
	if r.Core.logged {
		r.Unlock()
		return
	}
	lines := make([]string, 0, 16)
	// core stats
	for _, v := range r.Core.Tracker {
		if v.kind == statsKindLatency && v.associatedVal > 0 {
			v.Value /= v.associatedVal
		}
	}
	r.Core.Tracker[statUptimeLatency].Value = int64(time.Since(r.starttime) / time.Microsecond)

	b, err := jsoniter.Marshal(r.Core)

	// reset all the latency stats only
	for _, v := range r.Core.Tracker {
		if v.kind == statsKindLatency {
			v.Value = 0
			v.associatedVal = 0
		}
	}
	if err == nil {
		lines = append(lines, string(b))
	}
	// capacity
	if time.Since(r.timeUpdatedCapacity) >= *r.capUpdPeriod {
		runlru = r.updateCapacity()
		r.timeUpdatedCapacity = time.Now()
		for mpath, fsCapacity := range r.Capacity {
			b, err := jsoniter.Marshal(fsCapacity)
			if err == nil {
				lines = append(lines, mpath+": "+string(b))
			}
		}
	}

	// disk
	riostat := getiostatrunner()
	riostat.RLock()
	r.CPUidle = riostat.CPUidle
	for dev, iometrics := range riostat.Disk {
		r.Disk[dev] = iometrics
		if riostat.IsZeroUtil(dev) {
			continue // skip zeros
		}
		b, err := jsoniter.Marshal(r.Disk[dev])
		if err == nil {
			lines = append(lines, dev+": "+string(b))
		}

		stats := make([]metric, len(iometrics))
		idx := 0
		for k, v := range iometrics {
			stats[idx] = metric{statsd.Gauge, k, v}
			idx++
		}
		gettarget().statsdC.Send("iostat_"+dev, stats...)
	}
	riostat.RUnlock()

	lines = append(lines, fmt.Sprintf("CPU idle: %s%%", r.CPUidle))

	r.Core.logged = true
	r.Unlock()

	// log
	for _, ln := range lines {
		glog.Infoln(ln)
	}
	return
}

func (r *storstatsrunner) housekeep(runlru bool) {
	t := gettarget()

	if runlru && *r.lruEnabled {
		go t.runLRU()
	}

	// Run prefetch operation if there are items to be prefetched
	if len(t.prefetchQueue) > 0 {
		go t.doPrefetch()
	}

	// keep total log size below the configured max
	if time.Since(r.timeCheckedLogSizes) >= logsTotalSizeCheckTime {
		go r.removeLogs(*r.logMaxTotal)
		r.timeCheckedLogSizes = time.Now()
	}
}

func (r *storstatsrunner) removeLogs(maxtotal uint64) {
	logfinfos, err := ioutil.ReadDir(r.logDir)
	if err != nil {
		glog.Errorf("GC logs: cannot read log dir %s, err: %v", r.logDir, err)
		return // ignore error
	}
	// sample name dfc.ip-10-0-2-19.root.log.INFO.20180404-031540.2249
	var logtypes = []string{".INFO.", ".WARNING.", ".ERROR."}
	for _, logtype := range logtypes {
		var (
			tot   = int64(0)
			infos = make([]os.FileInfo, 0, len(logfinfos))
		)
		for _, logfi := range logfinfos {
			if logfi.IsDir() {
				continue
			}
			if !strings.Contains(logfi.Name(), ".log.") {
				continue
			}
			if strings.Contains(logfi.Name(), logtype) {
				tot += logfi.Size()
				infos = append(infos, logfi)
			}
		}
		if tot > int64(maxtotal) {
			if len(infos) <= 1 {
				glog.Errorf("GC logs: %s, total %d for type %s, max %d", r.logDir, tot, logtype, maxtotal)
				continue
			}
			r.removeOlderLogs(tot, int64(maxtotal), infos)
		}
	}
}

func (r *storstatsrunner) removeOlderLogs(tot, maxtotal int64, filteredInfos []os.FileInfo) {
	fiLess := func(i, j int) bool {
		return filteredInfos[i].ModTime().Before(filteredInfos[j].ModTime())
	}
	if glog.V(3) {
		glog.Infof("GC logs: started")
	}
	sort.Slice(filteredInfos, fiLess)
	for _, logfi := range filteredInfos[:len(filteredInfos)-1] { // except last = current
		logfqn := r.logDir + "/" + logfi.Name()
		if err := os.Remove(logfqn); err == nil {
			tot -= logfi.Size()
			glog.Infof("GC logs: removed %s", logfqn)
			if tot < maxtotal {
				break
			}
		} else {
			glog.Errorf("GC logs: failed to remove %s", logfqn)
		}
	}
	if glog.V(3) {
		glog.Infof("GC logs: done")
	}
}

func (r *storstatsrunner) updateCapacity() (runlru bool) {
	availableMountpaths, _ := fs.Mountpaths.Get()
	capacities := make(map[string]*fscapacity, len(availableMountpaths))

	for mpath := range availableMountpaths {
		statfs := &syscall.Statfs_t{}
		if err := syscall.Statfs(mpath, statfs); err != nil {
			glog.Errorf("Failed to statfs mp %q, err: %v", mpath, err)
			continue
		}
		fsCap := newFSCapacity(statfs)
		capacities[mpath] = fsCap
		if fsCap.Usedpct >= *r.lruHighWM {
			runlru = true
		}
	}

	r.Capacity = capacities
	return
}

func (r *storstatsrunner) doAdd(nv NamedVal64) {
	r.Lock()
	s := r.Core
	s.doAdd(nv.name, nv.val)
	r.Unlock()
}
