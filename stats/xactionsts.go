/*
 * Copyright (c) 2018, NVIDIA CORPORATION. All rights reserved.
 */
package stats

import (
	"time"

	"github.com/NVIDIA/dfcpub/cmn"
	"github.com/json-iterator/go"
)

type (
	XactionStatsRetriever interface {
		getStats([]XactionDetails) []byte
	}
	XactionStats struct {
		Kind        string                         `json:"kind"`
		TargetStats map[string]jsoniter.RawMessage `json:"target"`
	}
	XactionDetails struct {
		Id        int64     `json:"id"`
		StartTime time.Time `json:"startTime"`
		EndTime   time.Time `json:"endTime"`
		Status    string    `json:"status"`
	}
	RebalanceTargetStats struct {
		Xactions     []XactionDetails `json:"xactionDetails"`
		NumSentFiles int64            `json:"numSentFiles"`
		NumSentBytes int64            `json:"numSentBytes"`
		NumRecvFiles int64            `json:"numRecvFiles"`
		NumRecvBytes int64            `json:"numRecvBytes"`
	}
	RebalanceStats struct {
		Kind        string                          `json:"kind"`
		TargetStats map[string]RebalanceTargetStats `json:"target"`
	}
	PrefetchTargetStats struct {
		Xactions           []XactionDetails `json:"xactionDetails"`
		NumFilesPrefetched int64            `json:"numFilesPrefetched"`
		NumBytesPrefetched int64            `json:"numBytesPrefetched"`
	}
	PrefetchStats struct {
		Kind        string                   `json:"kind"`
		TargetStats map[string]PrefetchStats `json:"target"`
	}
)

func (p PrefetchTargetStats) getStats(allXactionDetails []XactionDetails) []byte {
	rstor := getstorstatsrunner()
	rstor.RLock()
	prefetchXactionStats := PrefetchTargetStats{
		Xactions:           allXactionDetails,
		NumBytesPrefetched: rstor.Core.Tracker[statPrefetchCount].Value,
		NumFilesPrefetched: rstor.Core.Tracker[statPrefetchSize].Value,
	}
	rstor.RUnlock()
	jsonBytes, err := jsoniter.Marshal(prefetchXactionStats)
	cmn.Assert(err == nil, err)
	return jsonBytes
}

func (r RebalanceTargetStats) getStats(allXactionDetails []XactionDetails) []byte {
	rstor := getstorstatsrunner()
	rstor.RLock()
	rebalanceXactionStats := RebalanceTargetStats{
		Xactions:     allXactionDetails,
		NumRecvBytes: rstor.Core.Tracker[statRxSize].Value,
		NumRecvFiles: rstor.Core.Tracker[statRxCount].Value,
		NumSentBytes: rstor.Core.Tracker[statTxSize].Value,
		NumSentFiles: rstor.Core.Tracker[statTxCount].Value,
	}
	rstor.RUnlock()
	jsonBytes, err := jsoniter.Marshal(rebalanceXactionStats)
	cmn.Assert(err == nil, err)
	return jsonBytes
}
