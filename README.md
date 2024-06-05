# uid池

## 简介

用作在特殊场景下(uid非完全自增生成)生成uid,如:不可生成特殊保留uid,但需要保证uid在除特殊保留uid外自增在且唯一

## 使用

```bash
$ go get github.com/leychan/uid-pool
```

### 清空池子,并起一个协程维护uid池子
```go
func loadUidPool() {
	cfg := uidpool.NewConfig(redis.MEMBER_UID_POOL_IN_REDIS, 1000, redis.MEMBER_UID_POOL_LOCK_IN_REDIS, config.GetRdb(), service.GetUidList, 
		uidpool.WithCronTimeDuration(1 * time.Second), uidpool.WithRetryTimeSleep(100 * time.Millisecond))
	uidpool.Conf = cfg
	uidpool.Flush() //每次启动都刷新,防止有重复的uid导致写库失败
	go uidpool.BgMaintain() //开启后台维护
}
```

### 获取uid
```go
import uidpool "github.com/leychan/uid-pool"

func GenerateUid() int {
    uid, _ := uidpool.GetUid()
    return uid
}
```
