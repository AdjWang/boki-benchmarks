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

    local req = wrk.format(method, path, headers, body)
    print("request:", req)
    return req
end

function dump(o)
   if type(o) == 'table' then
      local s = '{ '
      for k,v in pairs(o) do
         if type(k) ~= 'number' then k = '"'..k..'"' end
         s = s .. '['..k..'] = ' .. dump(v) .. ','
      end
      return s .. '} '
   else
      return tostring(o)
   end
end

response = function(status, headers, body)
  print("status:", status)
  print("headers:", dump(headers))
  print("body:", body)
  if status ~= 200 then
    error(status)
  end
end
