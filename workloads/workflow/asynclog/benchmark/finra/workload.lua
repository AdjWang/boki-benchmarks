request = function()
    local path = '/asyncFunction/fetchData'
    local method = "POST"
    local headers = {}
    local body = '{' ..
      '"InstanceId": "",' ..
      '"CallerName": "",' .. '"Async": true,' ..
      '"Input":{' ..
        '"Function":"fetchData",' ..
        '"Input": {' ..
          '"body": {' ..
            '"n_parallel": 1,' ..
            '"portfolioType": "S&P",' ..
            '"portfolio": "1234"' ..
          '}' ..
        '}' ..
      '}' ..
    '}'
    headers["Content-Type"] = "application/json"

    return wrk.format(method, path, headers, body)
end
