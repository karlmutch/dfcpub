/*
 * Copyright (c) 2018, NVIDIA CORPORATION. All rights reserved.
 */
package iostat

func GetFSUsedPercentage(path string) (usedPercentage uint64, ok bool) {
	totalBlocks, blocksAvailable, _, err := GetFSStats(path)
	if err != nil {
		return
	}
	usedBlocks := totalBlocks - blocksAvailable
	return usedBlocks * 100 / totalBlocks, true
}
