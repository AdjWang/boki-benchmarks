-- require "socket"
-- local JSON = require("JSON")
-- local UUID = require("uuid")
-- time = socket.gettime() * 1000
-- math.randomseed(time)
-- UUID.randomseed(time)

-- local function uuid()
--     return UUID():gsub('-', '')
-- end

request = function()
    local path = '/asyncFunction/singleop'
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

    return wrk.format(method, path, headers, body)
end

function init(args)
    math.randomseed(os.time())
end
