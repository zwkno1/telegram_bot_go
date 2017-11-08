
--[[
-- keys[1]: type:id
-- keys[2]: id
-- keys[3]: message
-- KEYS[4]: name
]]--

local rank_key = "rank:" .. KEYS[1] 
local ret = redis.call('zrevrange', rank_key, 0, 10, 'withscores')

local t = {}
local i = 0
for _, v in ipairs(ret) do
	i = i+1
	if (i%2 == 1) then
		local name  = redis.call('hget', 'user_name', v)
		if(name == nil) then 
			name = 'xxx'
		end
		if(name == '') then 
			name = 'xxx'
		end
		t[i] = name
	else
		t[i] = v
	end
end

return t



