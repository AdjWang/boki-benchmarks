// A super simple substitution of python yfinance functionalities used in
// marketdata/main.go
package yfinance

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

func request(requestURL string) ([]byte, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not create request")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "error making http request")
	}

	if res.StatusCode != 200 {
		return nil, errors.Wrapf(err, "error status code=%d", res.StatusCode)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read response body")
	}
	return resBody, nil
}

func parseClosingPrice(jsonData []byte) (float64, error) {
	var ticketData map[string]interface{}
	err := json.Unmarshal(jsonData, &ticketData)
	if err != nil {
		return 0.0, errors.Wrapf(err, "failed to unmarshal ticketData=%+v", jsonData)
	}
	// {
	//    "chart":{
	//       "result":[
	//          {
	//             "meta":{...},
	//             "timestamp":[...],
	//             "indicators":{
	//                "quote":[
	//                   {
	//                      "low":[...],
	//                      "high":[...],
	//                      "open":[...],
	//                      "close":[...],
	//                      "volume":[...]
	//                   }
	//                ]
	//             }
	//          }
	//       ],
	//       "error":null
	//    }
	// }
	chart := ticketData["chart"].(map[string]interface{})
	result := chart["result"].([]interface{})
	indicators := result[0].(map[string]interface{})["indicators"].(map[string]interface{})
	quote := indicators["quote"].([]interface{})
	close := quote[0].(map[string]interface{})["close"].([]interface{})
	return close[0].(float64), nil
}

func GetLastClosingPrice(ticker string) (float64, error) {
	requestURL := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?range=1", ticker)
	jsonData, err := request(requestURL)
	if err != nil {
		return 0.0, errors.Wrapf(err, "failed to request=%s", requestURL)
	}
	return parseClosingPrice(jsonData)
}
