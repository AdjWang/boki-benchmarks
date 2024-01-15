package yfinance

import (
	"io"
	"os"
	"testing"
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
