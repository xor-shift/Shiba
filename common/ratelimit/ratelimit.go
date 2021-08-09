package ratelimit

import "time"

type Bucket struct {
	maxTokens     int64
	msPerNewToken int64
	lastToken     time.Time
	tokens        int64
}

func NewBucket(maxTok int64, msPerTok int64) Bucket {
	return Bucket{
		maxTokens:     maxTok,
		msPerNewToken: msPerTok,
		lastToken:     time.Now(),
		tokens:        maxTok,
	}
}

//NewDrop returns whether or not the request can be fulfilled (i.e. it returns true if there is no overflow)
func (b *Bucket) NewDrop() bool {
	if now := time.Now(); b.lastToken.Before(now) {
		diff := now.Sub(b.lastToken)
		ms := diff.Milliseconds()
		nNewTokens := ms / b.msPerNewToken
		b.tokens += nNewTokens
		if b.tokens > b.maxTokens {
			b.tokens = b.maxTokens
		}
		passed := time.Duration(nNewTokens*b.msPerNewToken) * time.Millisecond
		b.lastToken = b.lastToken.Add(passed)
	}

	if b.tokens <= 0 {
		return false
	} else {
		b.tokens--
		return true
	}
}

type lruNode struct {
	mapKey     string
	moreRecent *lruNode
	lessRecent *lruNode
	bucket     Bucket
}

type RateLimiter struct {
	maxLRUNodes       int
	cacheBucketConfig Bucket
	globalBucket      Bucket
	lruMostRecent     *lruNode
	lruLeastRecent    *lruNode
	lruMap            map[string]*lruNode
}

func NewRateLimiter(userCacheSize int, globalMaxTokens int64, globalMSPerToken int64, userMaxTokens int64, userMSPerToken int64) *RateLimiter {
	r := new(RateLimiter)

	r.maxLRUNodes = userCacheSize
	r.cacheBucketConfig = NewBucket(userMaxTokens, userMSPerToken)
	r.globalBucket = NewBucket(globalMaxTokens, globalMSPerToken)
	r.lruMap = make(map[string]*lruNode)

	return r
}

func (r *RateLimiter) get(idx string) *lruNode {
	l, ok := r.lruMap[idx]
	if !ok {
		return nil
	}

	//node exists, yank it out
	if l.moreRecent != nil {
		l.moreRecent.lessRecent = l.lessRecent
	}

	if l.lessRecent != nil {
		l.lessRecent.moreRecent = l.moreRecent
	}

	if r.lruLeastRecent == l {
		r.lruLeastRecent = l.moreRecent
	}

	if r.lruMostRecent == l {
		r.lruMostRecent = l.lessRecent
	}

	l.moreRecent = nil
	l.lessRecent = nil

	//the node is now free standing, emplace it back to the front
	if r.lruMostRecent != nil {
		r.lruMostRecent.moreRecent = l
		l.lessRecent = r.lruMostRecent
	}
	r.lruMostRecent = l

	if r.lruLeastRecent == nil {
		r.lruLeastRecent = l
	}

	return l
}

func (r *RateLimiter) getOrEmplace(idx string) *lruNode {
	if l := r.get(idx); l != nil {
		return l
	}

	newNode := new(lruNode)
	newNode.mapKey = idx
	newNode.lessRecent = nil
	newNode.moreRecent = nil
	newNode.bucket = NewBucket(r.cacheBucketConfig.maxTokens, r.cacheBucketConfig.msPerNewToken)

	if r.lruMostRecent != nil {
		r.lruMostRecent.moreRecent = newNode
		newNode.lessRecent = r.lruMostRecent
	}
	r.lruMostRecent = newNode

	if r.lruLeastRecent == nil {
		r.lruLeastRecent = newNode
	}

	r.lruMap[idx] = r.lruMostRecent

	if len(r.lruMap) > r.maxLRUNodes {
		key := r.lruLeastRecent.mapKey
		r.lruLeastRecent = r.lruLeastRecent.moreRecent
		r.lruLeastRecent.lessRecent = nil
		delete(r.lruMap, key)
	}

	return r.lruMostRecent
}

func (r *RateLimiter) Check(key string) bool {
	if !r.globalBucket.NewDrop() {
		return false
	}

	l := r.getOrEmplace(key)

	return l.bucket.NewDrop()
}
