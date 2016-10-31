package lib

import (
	"sync"
	"time"
)

// LimitRate 有缺陷，见下面 TODO注释.
type LimitRate struct {
	rate       int
	interval   time.Duration
	lastAction time.Time
	lock       sync.Mutex
}

// Limit waits until the rate is under limit
func (l *LimitRate) Limit() bool {
	result := false
	for {
		l.lock.Lock()
		//判断最 后一次执行的时间 与 当前的时间间隔 是否大于限速速率
		// TODO 这里有缺点：1s内的请求只能均匀的到来，如瞬间来 N 个，N < rate, 那么只有一个能立刻返回，剩下的只能等待。
		if time.Now().Sub(l.lastAction) > l.interval {
			l.lastAction = time.Now()
			result = true
		}
		l.lock.Unlock()
		if result {
			return result
		}
		time.Sleep(l.interval)
	}
}

//SetRate 设置Rate
func (l *LimitRate) SetRate(r int) {
	l.rate = r
	l.interval = time.Second / time.Duration(l.rate)
}

//GetRate 获取Rate
func (l *LimitRate) GetRate() int {
	return l.rate
}
