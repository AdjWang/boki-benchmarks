package utils

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

const kLetterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const TimestampStrLen = 20
const SeqNumStrLen = 16

func SeqNumGenerator() chan string {
	seqCh := make(chan string, 100)
	go func() {
		seqNum := uint64(0)
		for {
			seqCh <- fmt.Sprintf("%016X", seqNum)
			seqNum++
		}
	}()
	return seqCh
}

func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = kLetterBytes[rand.Intn(len(kLetterBytes))]
	}
	return string(b)
}

func FormatTime(t time.Time) string {
	return fmt.Sprintf("%020d", t.UnixNano())
}

func ParseTime(payload string) time.Time {
	timeStr := payload[0:TimestampStrLen]
	if s, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
		return time.Unix(0, s)
	} else {
		panic(err)
	}
}

func ParseSeqNum(payload string) string {
	seqNumStr := payload[TimestampStrLen : TimestampStrLen+SeqNumStrLen]
	if s, err := strconv.ParseUint(seqNumStr, 16, 64); err == nil {
		return strconv.FormatUint(s, 10)
	} else {
		panic(err)
	}
}
