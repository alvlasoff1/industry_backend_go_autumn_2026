package main

import "container/list"

type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	ll       *list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		ll:       list.New(),
		items:    make(map[K]*list.Element, capacity),
	}
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	if c.capacity <= 0 {
		return *new(V), false
	}
	if item, ok := c.items[key]; ok {
		c.ll.MoveToFront(item)
		return item.Value.(entry[K, V]).value, true
	}
	return *new(V), false
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}
	if _, ok := c.items[key]; ok {
		c.items[key].Value = entry[K, V]{key, value}
		return
	}
	if len(c.items) == c.capacity {
		d := c.ll.Back()
		delete(c.items, d.Value.(entry[K, V]).key)
		c.ll.Remove(d)
	}
	c.items[key] = c.ll.PushFront(entry[K, V]{key, value})
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
