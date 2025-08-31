package mongodb

import (
	"container/heap"
	"sync"
	"time"

	"github.com/name5566/leaf/log"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Session 封装 mgo.Session 并增加引用计数和堆索引
type Session struct {
	*mgo.Session
	ref   int // 当前引用次数
	index int // 在堆中的索引
}

// SessionHeap 实现堆接口，用于管理 Session 按引用次数排序
type SessionHeap []*Session

func (h SessionHeap) Len() int {
	return len(h)
}

func (h SessionHeap) Less(i, j int) bool {
	return h[i].ref < h[j].ref // 引用次数少的优先
}

func (h SessionHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *SessionHeap) Push(s interface{}) {
	s.(*Session).index = len(*h)
	*h = append(*h, s.(*Session))
}

func (h *SessionHeap) Pop() interface{} {
	l := len(*h)
	s := (*h)[l-1]
	s.index = -1
	*h = (*h)[:l-1]
	return s
}

// DialContext 管理 Session 堆，保证 goroutine 安全
type DialContext struct {
	sync.Mutex
	sessions SessionHeap
}

// Dial 创建 DialContext，默认超时
func Dial(url string, sessionNum int) (*DialContext, error) {
	c, err := DialWithTimeout(url, sessionNum, 10*time.Second, 5*time.Minute)
	return c, err
}

// DialWithTimeout 创建 DialContext，并设置连接超时和同步超时
func DialWithTimeout(url string, sessionNum int, dialTimeout time.Duration, timeout time.Duration) (*DialContext, error) {
	if sessionNum <= 0 {
		sessionNum = 100
		log.Release("invalid sessionNum, reset to %v", sessionNum)
	}

	s, err := mgo.DialWithTimeout(url, dialTimeout)
	if err != nil {
		return nil, err
	}
	s.SetSyncTimeout(timeout)
	s.SetSocketTimeout(timeout)

	c := new(DialContext)

	// 初始化 session 堆
	c.sessions = make(SessionHeap, sessionNum)
	c.sessions[0] = &Session{s, 0, 0}
	for i := 1; i < sessionNum; i++ {
		c.sessions[i] = &Session{s.New(), 0, i}
	}
	heap.Init(&c.sessions)

	return c, nil
}

// Close 关闭所有 session，检查引用是否为 0
func (c *DialContext) Close() {
	c.Lock()
	for _, s := range c.sessions {
		s.Close()
		if s.ref != 0 {
			log.Error("session ref = %v", s.ref)
		}
	}
	c.Unlock()
}

// Ref 获取引用次数最少的 session 并增加引用计数
func (c *DialContext) Ref() *Session {
	c.Lock()
	s := c.sessions[0]
	if s.ref == 0 {
		s.Refresh() // 刷新 session
	}
	s.ref++
	heap.Fix(&c.sessions, 0) // 更新堆
	c.Unlock()

	return s
}

// UnRef 释放 session 的引用并调整堆顺序
func (c *DialContext) UnRef(s *Session) {
	c.Lock()
	s.ref--
	heap.Fix(&c.sessions, s.index)
	c.Unlock()
}

// EnsureCounter 确保某集合的计数器存在，如果不存在则创建
func (c *DialContext) EnsureCounter(db string, collection string, id string) error {
	s := c.Ref()
	defer c.UnRef(s)

	err := s.DB(db).C(collection).Insert(bson.M{
		"_id": id,
		"seq": 0,
	})
	if mgo.IsDup(err) {
		return nil
	} else {
		return err
	}
}

// NextSeq 获取计数器下一个值
func (c *DialContext) NextSeq(db string, collection string, id string) (int, error) {
	s := c.Ref()
	defer c.UnRef(s)

	var res struct {
		Seq int
	}
	_, err := s.DB(db).C(collection).FindId(id).Apply(mgo.Change{
		Update:    bson.M{"$inc": bson.M{"seq": 1}}, // 原子递增
		ReturnNew: true,
	}, &res)

	return res.Seq, err
}

// EnsureIndex 确保集合索引存在
func (c *DialContext) EnsureIndex(db string, collection string, key []string) error {
	s := c.Ref()
	defer c.UnRef(s)

	return s.DB(db).C(collection).EnsureIndex(mgo.Index{
		Key:    key,
		Unique: false,
		Sparse: true,
	})
}

// EnsureUniqueIndex 确保唯一索引存在
func (c *DialContext) EnsureUniqueIndex(db string, collection string, key []string) error {
	s := c.Ref()
	defer c.UnRef(s)

	return s.DB(db).C(collection).EnsureIndex(mgo.Index{
		Key:    key,
		Unique: true,
		Sparse: true,
	})
}
