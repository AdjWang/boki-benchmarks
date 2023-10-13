package sharedindex

import (
	"context"
	"errors"
	"log"

	"cs.utexas.edu/zjia/faas/ipc"
	"cs.utexas.edu/zjia/faas/protocol"
	"cs.utexas.edu/zjia/faas/types"
)

func TestBasicAlloc(ctx context.Context, faasEnv types.Environment) {
	tags := []uint64{2}
	data := []byte{1, 2, 3}
	seqNum, err := faasEnv.SharedLogAppend(ctx, tags, data)
	if err != nil {
		panic(err)
	}
	log.Printf("[DEBUG] test seqnum=%016X", seqNum)

	logSpaceId := uint32(seqNum >> 32)
	indexData, err := ipc.ConstructIndexData(0 /*metalogProgress*/, logSpaceId)
	if err != nil {
		panic(err)
	}

	{
		metaLogProgress, resultSeqNum, err := indexData.IndexReadPrev(0 /*metaLogProgress*/, seqNum, uint64(2) /*tag*/)
		if err != nil {
			panic(err)
		}
		log.Printf("[DEBUG] metaLogProgress=%016X, resultSeqNum=%016X", metaLogProgress, resultSeqNum)
	}

	{
		metaLogProgress, resultSeqNum, err := indexData.IndexReadPrev(0 /*metaLogProgress*/, seqNum, uint64(3) /*tag*/)
		if errors.Is(err, ipc.Err_Empty) {
			log.Printf("[DEBUG] empty")
		} else {
			// expect find nothing
			log.Panicf("[FATAL] metaLogProgress=%016X, resultSeqNum=%016X", metaLogProgress, resultSeqNum)
		}
	}

	{
		message, err := indexData.LogReadPrev(0 /*metaLogProgress*/, seqNum, uint64(2) /*tag*/)
		if err != nil {
			panic(err)
		}
		metaLogProgress := protocol.GetMetalogProgressInMessage(message)
		seqNum := protocol.GetLogSeqNumFromMessage(message)
		log.Printf("[DEBUG] metaLogProgress=%016X, resultSeqNum=%016X", metaLogProgress, seqNum)
	}

	indexData.Uninstall()
}
