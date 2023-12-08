package dbops

import (
	"github.com/eniac/Beldi/internal/txnbench/internal/common"
	"github.com/eniac/Beldi/pkg/cayonlib"
	"github.com/mitchellh/mapstructure"
)

func DBRead(env *cayonlib.Env, table string, key string) bool {
	ok, item := cayonlib.TPLRead(env, table, key)
	if !ok {
		return false
	}
	var data common.ReadOnlyData
	cayonlib.CHECK(mapstructure.Decode(item, &data))
	if len(data.ByteStream) != common.DataSize {
		return false
	}
	return true
}
