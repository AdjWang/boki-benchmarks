package main

import (
	"fmt"
	"time"

	"github.com/go-zookeeper/zk"
)

func mustExists(zkcli *zk.Conn, node string) {
	ok, _, err := zkcli.Exists(node)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic(fmt.Errorf("node: %v not exist", node))
	}
}

func main() {
	zkcli, _, err := zk.Connect([]string{"zookeeper"}, time.Second, zk.WithLogInfo(false))
	if err != nil {
		panic(err)
	}
	mustExists(zkcli, "/faas")
	mustExists(zkcli, "/faas/node")
	mustExists(zkcli, "/faas/view")
	mustExists(zkcli, "/faas/freeze")
	mustExists(zkcli, "/faas/cmd")
}
