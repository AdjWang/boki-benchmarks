package dbops

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/eniac/Beldi/internal/txnbench/internal/common"
	"github.com/eniac/Beldi/pkg/cayonlib"
)

func DBWrite(env *cayonlib.Env, table string, key string) bool {
	data := common.RandomString(common.DataSize)
	ok := cayonlib.TPLWrite(env, table, key,
		aws.JSONValue{"V.ByteStream": data})
	return ok
}
