package sharedindex

import (
	"context"

	"cs.utexas.edu/zjia/faas/ipc"
	"cs.utexas.edu/zjia/faas/types"
)

func TestBasicBinding(ctx context.Context, faasEnv types.Environment) {
	ipc.TestBinding()
}
