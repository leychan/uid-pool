package uidpool

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var Conf *Config

type Config struct {
    CronTimeDuration time.Duration //定时维护池子执行间隔
    RetryTimes int // 重试次数
    RetryTimeSleep time.Duration // 重试间隔
    CacheKey string  //缓存key, 必须设置
    Threshold int  //池子阈值,低于此值将触发填充
    LockKey string //执行填充的时候,需要先加锁
    Rdb *redis.Client //redis client
    GetUidList func() ([]int) //获取uid列表方法
}

// GetUid 获取uid
func GetUid() (int, error) {
    for i := 0; i < Conf.RetryTimes; i++ {
        uid, _ := getUid()
        if uid != 0 {
            return uid, nil
        }
        time.Sleep(Conf.RetryTimeSleep)
    }
    return 0, fmt.Errorf("get uid failed")
}

func getUid() (int, error) {
    // 从redis中获取一个uid
    uid, err := Conf.Rdb.RPop(context.Background(), Conf.CacheKey).Int()
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
    return Conf.Rdb.RPop(context.Background(), Conf.CacheKey).Int()
}

// Maintain 维护uid池子
func maintain() error {
    //获取池子长度
    l, err := Conf.Rdb.LLen(context.Background(), Conf.CacheKey).Result()
    //执行出错返回错误
    if err != redis.Nil && err != nil {
        return err
    }
    //长度大于阈值, 直接返回
    if l > int64(Conf.Threshold) {
        return nil
    }
    //尝试加锁,如果获取锁失败,返回错误,可忽略
    if !lock() {
        return fmt.Errorf("lock failed")
    }
    defer Unlock()
    //再次获取
    uidList := Conf.GetUidList()

    return fillUidPool(uidList)
}

// Lock 加锁,防止重复填充
func lock() bool {
    set, err := Conf.Rdb.SetNX(context.Background(), Conf.LockKey, 1, time.Second * 60).Result()
    if err != nil {
        return false
    }
    return set
}

func Unlock() error {
    _, err := Conf.Rdb.Del(context.Background(), Conf.LockKey).Result()
    return err
}

// Flush 清空池子
func Flush() error {
    _, err := Conf.Rdb.Del(context.Background(), Conf.CacheKey).Result()
    return err
}

// BgMaintain 定时维护池子
func BgMaintain() {
    for {
        maintain()
        time.Sleep(Conf.CronTimeDuration)
    }
}

// FillUidPool 填充uid池子
func fillUidPool(uidList []int) error {
    pipeline := Conf.Rdb.Pipeline()

    for _, uid := range uidList {
        pipeline.LPush(context.Background(), Conf.CacheKey, uid)
    }

    pipeline.Exec(context.Background())
    return nil
}