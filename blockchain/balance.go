package blockchain

import (
	"container/ring"
	"sync"
)

var factories map[string]factory = map[string]factory{
	"round-robin": newRoundRobin,
}

type backend struct {
	hostname string
}

func (b *backend) String() string {
	return b.hostname
}

type backender interface {
	String() string
}

type backends interface {
	Choose() backender
	Len() int
	Add(string)
	Remove(string)
}

type factory func([]string) backends

func build(algorithm string, specs []string) backends {
	factory, found := factories[algorithm]
	if !found {
		blockchainLogger.Infof("balance algorithm %s not supported", algorithm)
		return build("round-robin", specs)
	}
	return factory(specs)
}

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
