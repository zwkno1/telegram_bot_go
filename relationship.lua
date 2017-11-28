
--[[
-- keys[1]: id
]]--

local rank_key = "relationship:" .. KEYS[1] 
local ret = redis.call('zrevrange', rank_key, 0, 10, 'withscores')

local t = {}
local i = 0
for _, v in ipairs(ret) do
	i = i+1
	if (i%2 == 1) then
		local name  = redis.call('hget', 'user_name', v)
		if((name == nil) or (name == 'xxx')) then 
			name = 'xxx'
		end
		t[i] = name
	else
		t[i] = v
	end
end

return t



