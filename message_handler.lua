--[[
-- keys[1]: type:id
-- keys[2]: id
-- keys[3]: message
-- KEYS[4]: name
]]--

local rank_key = "rank:" .. KEYS[1] 
local message_key = "message:" .. KEYS[1] .. ":" .. KEYS[2]

local last_key = 'last_message:'..KEYS[1]
local last = redis.call('get', last_key)
if (last == nil or last ~= KEYS[2]) then
	redis.call('zincrby', rank_key, 1, KEYS[2])
	redis.call('set', last_key, KEYS[2])
end

redis.call('rpush', message_key, KEYS[3])
redis.call('hset', 'user_name', KEYS[2], KEYS[4])

local text_rank_key = "textrank:" .. KEYS[1]

for i=1, #ARGV, 1 do
	redis.call('zincrby', text_rank_key, 1, ARGV[i])
end

return 'ok'

