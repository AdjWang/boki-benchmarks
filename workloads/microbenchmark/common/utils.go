package common

import (
	"errors"
	"fmt"
)

func AssertSeqNumOrder(seqNums []uint64) error {
	if len(seqNums) == 0 {
		return nil
	}
	for i := 0; i < len(seqNums)-1; i++ {
		if seqNums[i] >= seqNums[i+1] {
			return errors.New(fmt.Sprintf("seqnum order assertion failed: (%d:%d, %d:%d)\n", i, seqNums[i], i+1, seqNums[i+1]))
		}
	}
	return nil
}

func ListSub(l1 []float64, l2 []float64) []float64 {
	if len(l1) != len(l2) {
		panic(fmt.Errorf("Inconsistent length: len1=%v len2=%v", len(l1), len(l2)))
	}
	result := make([]float64, len(l1))
	for i := 0; i < len(l1); i++ {
		result[i] = l1[i] - l2[i]
	}
	return result
}
