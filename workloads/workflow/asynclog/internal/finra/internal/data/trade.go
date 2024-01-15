package data

import "encoding/json"

type Trade struct {
	Security   string      `json:"Security"`
	LastQty    json.Number `json:"LastQty"`
	LastPx     json.Number `json:"LastPx"`
	Side       json.Number `json:"Side"`
	TrdSubType json.Number `json:"TrdSubType"`
	TradeDate  string      `json:"TradeDate"`
}

const PortfoliosData string = `
{
	"1234" : [
        {
            "Security": "GOOG", 
            "LastQty" : 10,
            "LastPx"  : 1363.85123,
            "Side"    : 1,
            "TrdSubType" : 0,
            "TradeDate" : "200507"
        },
        {
            "Security": "MSFT", 
            "LastQty" : 20,
            "LastPx"  : 183.851234,
            "Side"    : 1,
            "TrdSubType" : 0,
            "TradeDate" : "200507"
        }
    ]
}
`

func ReadPortfolios() (map[string][]Trade, error) {
	var portfolios map[string][]Trade
	err := json.Unmarshal([]byte(PortfoliosData), &portfolios)
	return portfolios, err
}
