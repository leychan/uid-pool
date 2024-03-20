package uidpool

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var conf Config

type Config struct {
    RetryTimes int // 重试次数
    RetryTimeSleep time.Duration
    CacheKey string  //缓存key, 必须设置
    Threshold int  //池子阈值,低于此值将触发填充
    LockKey string //执行填充的时候,需要先加锁
    Rdb *redis.Client //redis client
    getUidList func() ([]int) //获取uid列表方法
}

// GetUid 获取uid
func GetUid() (int, error) {
    for i := 0; i < conf.RetryTimes; i++ {
        uid, _ := getUid()
        if uid != 0 {
            return uid, nil
        }
        time.Sleep(conf.RetryTimeSleep)
    }
    return 0, fmt.Errorf("get uid failed")
}

func getUid() (int, error) {
    // 从redis中获取一个uid
    uid, err := conf.Rdb.RPop(context.Background(), conf.CacheKey).Int()
    //出错返回0和error
    if err != redis.Nil && err != nil {
        return 0, err
    }

    //正常获取到uid, 返回
    if uid != 0 {
        return uid, nil
    }
    // 没有获取到uid, 则尝试更新池子
    err = maintain()
    if err != nil {
        return 0, err
    }
    return conf.Rdb.RPop(context.Background(), conf.CacheKey).Int()
}

// Maintain 维护uid池子
func maintain() error {
    //获取池子长度
    l, err := conf.Rdb.LLen(context.Background(), conf.CacheKey).Result()
    //执行出错返回错误
    if err != redis.Nil && err != nil {
        return err
    }
    //长度大于阈值, 直接返回
    if l > int64(conf.Threshold) {
        return nil
    }
    //尝试加锁,如果获取锁失败,返回错误,可忽略
    if !lock() {
        return fmt.Errorf("lock failed")
    }
    //再次获取
    uidList := conf.getUidList()

    return fillUidPool(uidList)
}

// Lock 加锁,防止重复填充
func lock() bool {
    set, err := conf.Rdb.SetNX(context.Background(), conf.LockKey, 1, time.Second * 60).Result()
    if err != nil {
        return false
    }
    return set
}

// Flush 清空池子
func Flush() error {
    _, err := conf.Rdb.Del(context.Background(), conf.CacheKey).Result()
    return err
}

// BgMaintain 定时维护池子
func BgMaintain() {
    for {
        maintain()
        time.Sleep(time.Second * 5)
    }
}

// FillUidPool 填充uid池子
func fillUidPool(uidList []int) error {
    pipeline := conf.Rdb.Pipeline()

    for _, uid := range uidList {
        pipeline.LPush(context.Background(), conf.CacheKey, uid)
    }

    pipeline.Exec(context.Background())
    return nil
}