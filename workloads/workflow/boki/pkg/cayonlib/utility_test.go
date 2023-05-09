package cayonlib

import (
	"log"
	"strconv"
	"testing"
	"time"
)

func TestTracerSerialize(t *testing.T) {
	tracer := NewLogTracer()

	tracer.TraceStart()
	time.Sleep(time.Second)
	tracer.TraceEnd()

	data, err := tracer.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	log.Println(string(data))
	logTracer, err := DeserializeLogTracer(data)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("[DEBUG] tracer: %v", logTracer.DebugString())

	rawTime := logTracer.LogString()
	intTime, err := strconv.ParseInt(rawTime, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	logTracer.TraceAdd(time.Duration(intTime) * time.Microsecond)
	log.Printf("[DEBUG] tracer: %v", logTracer.DebugString())
}
