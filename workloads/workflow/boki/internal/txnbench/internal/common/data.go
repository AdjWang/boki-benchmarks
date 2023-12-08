package common

import "math/rand"

// const DataSize int = 10 // bytes
const DataSize int = 1024 // bytes

func RandomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

const FnReadOnly string = "readonly"
const FnWriteOnly string = "writeonly"
const FnDBOps string = "dbops"
const FnDBReadOnly string = "dbread"
const FnDBWriteOnly string = "dbwrite"

const TableReadOnly string = "readonly"
const TableWriteOnly string = "writeonly"
const DefaultKey string = "ByteStream"
