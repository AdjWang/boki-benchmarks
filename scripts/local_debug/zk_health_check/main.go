package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/pkg/errors"
)

func mustExists(zkcli *zk.Conn, node string) {
	ok, _, err := zkcli.Exists(node)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic(fmt.Errorf("[FATAL] node=%v not exist", node))
	}
}

// inspect unhealthy log:
// docker inspect --format "{{json .State.Health }}" $(docker ps -a | grep unhealthy | awk '{print $1}') | jq
func main() {
	log.Println("[INFO] health checking...")
	zkcli, _, err := zk.Connect([]string{"zookeeper"}, time.Second, zk.WithLogInfo(false))
	if err != nil {
		panic(errors.Wrap(err, "[FATAL] zookeeper connect failed"))
	}
	mustExists(zkcli, "/faas")
	mustExists(zkcli, "/faas/node")
	mustExists(zkcli, "/faas/view")
	mustExists(zkcli, "/faas/freeze")
	mustExists(zkcli, "/faas/cmd")
}
