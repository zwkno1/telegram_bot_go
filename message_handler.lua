--[[
-- keys[1]: type:id
-- keys[2]: id
-- keys[3]: message
-- KEYS[4]: name
-- KEYS[5]: reply to id
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

-- text rank
local text_rank_key = "textrank:" .. KEYS[1]
for i=1, #ARGV, 1 do
        redis.call('zincrby', text_rank_key, 1, ARGV[i])
end

-- relationship
local last10_key = 'last10:' .. KEYS[1]
local scores = { 10, 9, 8, 7, 6, 5, 4, 3, 2, 1 }
local last10 = redis.call('lrange', last10_key, 0, 9)

if (KEYS[5] ~= '0') then
        local key = 'relationship:' .. KEYS[5]
        redis.call('zincrby', key, 50, KEYS[2])
end

local k = {}
for i=1, #last10, 1 do
        if ((last10[i] ~= KEYS[2]) and (k[last10[i]] == nil)) then
                local relationship_key = 'relationship:' .. last10[i]
                redis.call('zincrby', relationship_key, scores[i], KEYS[2])
                k[last10[i]] = 1
        end
end

redis.call('lpush', last10_key, KEYS[2])
if (#last10 == 10) then
        redis.call('rpop', last10_key)
end

return 'ok'

