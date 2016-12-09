package blockchain

import (
	"container/ring"
	"sync"
)

// 负载均衡工厂
var factories map[string]factory = map[string]factory{
	"round-robin": newRoundRobin,
}

// 后端服务对象
type backend struct {
	hostname string
}

func (b *backend) String() string {
	return b.hostname
}

type backender interface {
	String() string
}

// 负载均衡器接口
type backends interface {
	// 选择一个后端服务信息
	Choose() backender
	// 统计元素长度
	Len() int
	// 添加一个服务信息元素
	Add(string)
	// 移除一个服务信息元素
	Remove(string)
}

// 负载均衡器工厂函数
type factory func([]string) backends

// 根据负载均衡算法签名，获取对应的负载均衡器
func build(algorithm string, specs []string) backends {
	factory, found := factories[algorithm]
	if !found {
		blockchainLogger.Infof("balance algorithm %s not supported", algorithm)
		return build("round-robin", specs)
	}
	return factory(specs)
}

// 轮询负载器
type roundRobin struct {
	r *ring.Ring
	l sync.RWMutex
}

func newRoundRobin(strs []string) backends {
	r := ring.New(len(strs))
	for _, s := range strs {
		r.Value = &backend{s}
		r = r.Next()
	}
	return &roundRobin{r: r}
}

func (self *roundRobin) Len() int {
	self.l.RLock()
	defer self.l.RUnlock()
	return self.r.Len()
}

func (self *roundRobin) Choose() backender {
	self.l.Lock()
	defer self.l.Unlock()
	if self.r == nil {
		return nil
	}
	n := self.r.Value.(*backend)
	self.r = self.r.Next()
	return n
}

func (self *roundRobin) Add(s string) {
	self.l.Lock()
	defer self.l.Unlock()
	nr := &ring.Ring{Value: &backend{s}}
	if self.r == nil {
		self.r = nr
	} else {
		self.r = self.r.Link(nr).Next()
	}
}

func (self *roundRobin) Remove(s string) {
	self.l.Lock()
	defer self.l.Unlock()
	r := self.r
	if self.r.Len() == 1 {
		self.r = ring.New(0)
		return
	}

	for i := self.r.Len(); i > 0; i-- {
		r = r.Next()
		ba := r.Value.(*backend)
		if s == ba.String() {
			self.r = r.Unlink(1)
			return
		}
	}
}
