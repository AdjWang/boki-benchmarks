package common

import (
	"errors"
	"fmt"
)

func AssertSeqNumOrder(res *FnOutput) error {
	seqNums := res.SeqNums
	for i := 0; i < len(seqNums)-1; i++ {
		if seqNums[i] >= seqNums[i+1] {
			return errors.New(fmt.Sprintf("seqnum order assertion failed: (%d:%d, %d:%d)\n", i, seqNums[i], i+1, seqNums[i+1]))
		}
	}
	return nil
}
