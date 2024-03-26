# uid池

## 简介

用作在特殊场景下(uid非完全自增生成)生成uid,如:不可生成特殊保留uid,但需要保证uid在除特殊保留uid外自增在且唯一

## 使用

```bash
$ go get github.com/leychan/uid-pool
```

### 清空池子,并起一个协程维护uid池子
```go
uidpool.Conf = &uidpool.Config{
    CronTimeDuration: time.Duration(5) * time.Second,
    RetryTimes:  3, // 从池子获取uid的重试次数
    RetryTimeSleep: time.Millisecond * 10, // 从池子获取uid的重试间隔
    CacheKey: "your_redis_key", // uid池子在redis的key
    LockKey:   "your_redis_lock_key",
    Threshold: 1000, // 池子长度低于此值时，执行维护
    Rdb:       redis.NewClient(&redis.Options{}), //redis客户端
    GetUidList: service.GetUidList, //补充获取uid的方法
}
//清空当前池子
uidpool.Flush()
//起一个协程维护池子
go uidpool.BgMaintain()
```

### 获取uid
```go
import uidpool "github.com/leychan/uid-pool"

func GenerateUid() int {
    uid, _ := uidpool.GetUid()
    return uid
}
```
