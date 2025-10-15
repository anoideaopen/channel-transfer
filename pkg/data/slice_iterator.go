package data

import "sync"

var _ Iterator = &SliceIterator{}

// SliceIterator implements a simple iterator based on slice iteration. It
// can be used in various storage implementations and provides simple iteration
// mechanism.
type SliceIterator struct {
	items []Object
	cur   int

	mu sync.Mutex
}

// NewSliceIterator creates a new object iterator based on slice iteration.
func NewSliceIterator(obj ...Object) *SliceIterator {
	objects := make([]Object, 0, len(obj))
	objects = append(objects, obj...)

	return &SliceIterator{
		items: objects,
	}
}

// HasNext returns true if it is possible to get the next object from the
// collection.
func (i *SliceIterator) HasNext() (out bool) {
	i.mu.Lock()
	out = i.cur != len(i.items)
	i.mu.Unlock()

	return out
}

// Next returns the next object from the iterator. This object is already
// unmarshaled and can be converted using type casting.
func (i *SliceIterator) Next() (Object, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.cur == len(i.items) {
		return nil, ErrObjectNotFound
	}

	obj := i.items[i.cur]
	i.cur++

	return obj, nil
}

// Close should be called every time the collection is finished. This method
// frees the iterator data in memory.
func (i *SliceIterator) Close() error {
	i.mu.Lock()
	i.cur = 0
	i.items = nil
	i.mu.Unlock()

	return nil
}
