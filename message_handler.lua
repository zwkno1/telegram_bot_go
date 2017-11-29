--[[
-- keys[1]: type:id
-- keys[2]: id
-- keys[3]: message
-- KEYS[4]: username
-- KEYS[5]: nickname
]]--

local rank_key = "rank:" .. KEYS[1] 
local message_key = "message:" .. KEYS[1] .. ":" .. KEYS[2]

-- rank
local last_key = 'last_message:'..KEYS[1]
local last = redis.call('get', last_key)
if (last == nil or last ~= KEYS[2]) then
	redis.call('zincrby', rank_key, 1, KEYS[2])
	redis.call('set', last_key, KEYS[2])
end

-- store message
redis.call('rpush', message_key, KEYS[3])

-- user id -> name map
redis.call('hset', 'user_name', KEYS[2], KEYS[4])
redis.call('hset', 'name_user', KEYS[4], KEYS[2])
redis.call('hset', 'user_nickname', KEYS[2], KEYS[5])

-- text rank
local text_rank_key = 'textrank:' .. KEYS[1]
for i=1, #ARGV, 1 do
	local is_forbid = redis.call('sismember', 'banned_words', ARGV[i])
	if (is_forbid == 0) then
		redis.call('zincrby', text_rank_key, 1, ARGV[i])
	end
end

return 'ok'

