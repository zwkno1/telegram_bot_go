--[[
-- keys[1]: type:id
-- keys[2]: id
-- KEYS[3]: reply id
-- ARGS[...]: at usernames
]]--

local max_message_num = 10

local scores = {}
for i=1, max_message_num do
	scores[i] =  11 - i
end
local reply_score = 50
local at_score = 100

-- at message
local ret  = "atusers: "
for i=1, #ARGV, 1 do
	local id  = redis.call('hget', 'name_user', ARGV[i])
	if (id) then
		local relationship_key = 'relationship:'..id
		redis.call('zincrby', relationship_key, at_score, KEYS[2])
	end
end

-- reply message
if (KEYS[5] ~= '0') then
        local key = 'relationship:' .. KEYS[3]
        redis.call('zincrby', key, reply_score, KEYS[2])
end

-- normal message
local last_message_key = 'last_10:' .. KEYS[1]
local k = {}
local last_message = redis.call('lrange', last_message_key, 0, max_message_num)
for i=1, #last_message, 1 do
        if ((last_message[i] ~= KEYS[2]) and (k[last_message[i]] == nil)) then
                local relationship_key = 'relationship:' .. last_message[i]
                redis.call('zincrby', relationship_key, scores[i], KEYS[2])
                k[last_message[i]] = 1
        end
end

-- update last_message
redis.call('lpush', last_message_key, KEYS[2])
if (#last_message == max_message_num) then
        redis.call('rpop', last_message_key)
end

return 'ok'

