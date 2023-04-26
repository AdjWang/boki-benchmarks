-- require "socket"
-- local JSON = require("JSON")
-- local UUID = require("uuid")
-- time = socket.gettime() * 1000
-- math.randomseed(time)
-- UUID.randomseed(time)

-- local function uuid()
--     return UUID():gsub('-', '')
-- end

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

request = function()
    local path = os.getenv("ENDPOINT")
    local method = "POST"
    local headers = {}
    -- local param = {
    --     InstanceId = uuid(),
    --     CallerName = "",
    --     Async = true,
    --     Input = {
    --         Type = os.getenv("TYPE"),
    --     }
    -- }
    local body = '{' ..
    --   '"InstanceId": "' .. uuid() .. '",' ..
      '"InstanceId": "",' ..
      '"CallerName": "",' .. '"Async": true,' ..
      '"Input":{}' ..
    '}'
    -- local body = JSON:encode(param)
    headers["Content-Type"] = "application/json"

    local req = wrk.format(method, path, headers, body)
    -- print(req)
    return req
end

-- function init(args)
--     math.randomseed(os.time())
-- end

function response(status, headers, body)
  -- print(status)
  -- print(dump(headers))
  io.stderr:write(body)
  io.stderr:write('\n')
end
