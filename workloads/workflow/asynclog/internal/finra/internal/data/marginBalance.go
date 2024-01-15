package data

import "encoding/json"

const MarginBalanceData string = `
{
	"1234" : 4500
}
`

func ReadMarginBalance() (map[string]int64, error) {
	var marginBalance map[string]int64
	err := json.Unmarshal([]byte(MarginBalanceData), &marginBalance)
	return marginBalance, err
}
