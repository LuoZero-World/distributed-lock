package lock

const LuaCheckAndDeleteDistributionLock = `
	local lockerKey = KEYS[1]
	local targetToken = ARGV[1]
	local getToken = redis.call('get', lockerKey)
	if(not getToken or getToken ~= targetToken) then
		return 0
	else
		return redis.call('del', lockerKey)
	end
`
