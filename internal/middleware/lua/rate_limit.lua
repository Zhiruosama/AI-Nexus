-- KEY[1]: 要进行限流的键
-- ARGV[1]: 时间窗口的大小
--
-- 返回值: 当前键的计数值

local current = redis.call('INCR', KEYS[1])

if tonumber(current) == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end

return current