package yfinance

import (
	"io"
	"os"
	"testing"
	"time"
)

func TestParseMarketdata(t *testing.T) {
	fd, err := os.Open("goog.json")
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()
	jsonData, err := io.ReadAll(fd)
	if err != nil {
		t.Fatal(err)
	}
	close, err := parseClosingPrice(jsonData)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(close)
}

func TestRequest(t *testing.T) {
	ts := time.Now()
	price, err := GetLastClosingPrice("GOOG")
	duration := time.Since(ts).Microseconds()
	t.Logf("[STAT] request duration=%d us", duration)

	if err != nil {
		t.Fatal(err)
	}
	t.Log(price)
}
