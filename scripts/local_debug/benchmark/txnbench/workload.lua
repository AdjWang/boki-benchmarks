-- require "socket"
-- local JSON = require("JSON")
-- local UUID = require("uuid")
-- time = socket.gettime() * 1000
-- UUID.randomseed(time)
-- math.randomseed(socket.gettime() * 1000)
math.random();
math.random();
math.random()

-- local function uuid()
--     return UUID():gsub('-', '')
-- end

-- local gatewayPath = os.getenv("ENDPOINT")
local baseline_prefix = ''

local function readonly()
    local method = "POST"
    local path = '/asyncFunction/'..baseline_prefix..'readonly'
    -- local path = '/function/'..baseline_prefix..'readonly'

    local body = '{' ..
    --   '"InstanceId": "' .. uuid() .. '",' ..
      '"InstanceId": "",' ..
      '"CallerName": "",' .. '"Async": true,' ..
      '"Input":{' ..
        '"Function":"readonly",' ..
        '"Input": {' ..
          '"table": "' .. 'readonly' .. '",' ..
          '"key": "' .. 'ByteStream' .. '"' ..
        '}' ..
      '}' ..
    '}'

    local headers = {}
    headers["Content-Type"] = "application/json"
    return wrk.format(method, path, headers, body)
end

local function writeonly()
    local method = "POST"
    local path = '/asyncFunction/'..baseline_prefix..'writeonly'
    -- local path = '/function/'..baseline_prefix..'writeonly'

    local body = '{' ..
    --   '"InstanceId": "' .. uuid() .. '",' ..
      '"InstanceId": "",' ..
      '"CallerName": "",' .. '"Async": true,' ..
      '"Input":{' ..
        '"Function":"writeonly",' ..
        '"Input": {' ..
          '"table": "' .. 'writeonly' .. '",' ..
          '"key": "' .. 'ByteStream' .. '"' ..
        '}' ..
      '}' ..
    '}'

    local headers = {}
    headers["Content-Type"] = "application/json"
    return wrk.format(method, path, headers, body)
end

init = function(args)
    local baseline = os.getenv("BASELINE")
    if baseline == '1' then
        print("benchmarking beldi baseline")
        baseline_prefix = 'b'
    else
        print("benchmarking beldi")
        baseline_prefix = ''
    end
end

local req_count = 1

request = function()
    -- cur_time = math.floor(socket.gettime())
    -- local search_ratio = 0.6
    -- local recommend_ratio = 0.39
    -- local user_ratio = 0.005
    -- local reserve_ratio = 0.005

    -- DEBUG
    local readonly_ratio = 0.5
    local writeonly_ratio = 0.5

    -- req = readonly()
    -- req = writeonly()
    local coin = math.random()
    local req
    if coin < readonly_ratio then
        req = readonly()
    else
        req = writeonly()
    end
    print("request:", req_count, req)
    req_count = req_count + 1
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
