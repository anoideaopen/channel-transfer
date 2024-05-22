package inmem

import (
	"context"
	"path"
	"strings"
	"sync"

	"github.com/anoideaopen/channel-transfer/pkg/data"
)

var _ data.Storage = &Storage{}

// Storage implements a data.Storage interface using a map stored in memory.
type Storage struct {
	data map[string][]byte
	m    sync.RWMutex
}

// NewStorage creates an instance of the Storage structure and initializes the
// embedded map.
func NewStorage() *Storage {
	return &Storage{
		data: make(map[string][]byte),
	}
}

// Save stores the object in a storage, using Key as the unique name (path)
// to save the object and Type (by calling Instance method) to share the
// namespace between objects. Key can be repeated for different Type values.
// The MarshalBinary method can be called before the object is saved.
func (s *Storage) Save(
	_ context.Context,
	obj data.Object,
	key data.Key,
) error {
	bin, err := obj.MarshalBinary()
	if err != nil {
		return err
	}

	s.m.Lock()
	s.data[s.path(obj.Instance(), key)] = bin
	s.m.Unlock()

	return nil
}

// Load loads an object from the repository, using Key as a unique name
// (path) to find the object and Type (by calling the Instance method). The
// UnmarshalBinary method can be called for the object. To restore an
// object, you must allocate memory for it to avoid panic when working with
// a nil address.
// If the object is not found, an ErrObjectNotFound error will be returned.
func (s *Storage) Load(
	_ context.Context,
	obj data.Object,
	key data.Key,
) error {
	s.m.RLock()
	bin, ok := s.data[s.path(obj.Instance(), key)]
	s.m.RUnlock()

	if !ok {
		return data.ErrObjectNotFound
	}

	return obj.UnmarshalBinary(bin)
}

// Search searches for objects in the repository of a certain Type. It
// returns a list of found objects or ErrObjectNotFound if no objects
// matching the criteria were found. Prefix is used for additional filtering
// of objects according to their Key with partial match at the beginning of
// the key. If you want to get all objects Prefix can be empty. The original
// object is only needed as a template, other instances will be created
// based on it (by method Copy), but the object itself will not be changed.
func (s *Storage) Search(
	_ context.Context,
	tmpl data.Object,
	pref data.Prefix,
) ([]data.Object, error) {
	out := make([]data.Object, 0, len(s.data))

	s.m.RLock()
	defer s.m.RUnlock()

	for p, d := range s.data {
		if !strings.HasPrefix(p, s.pref(tmpl.Instance(), pref)) {
			continue
		}

		clone := tmpl.Clone()
		if err := clone.UnmarshalBinary(d); err != nil {
			return nil, err
		}

		out = append(out, clone)
	}

	if len(out) == 0 {
		return nil, data.ErrObjectNotFound
	}

	return out, nil
}

// Iterate does what Search does, but in place of the finished collection of
// objects an iterator will be returned that can be used to list the objects
// one by one. The original object is only needed as a template, other
// instances will be created based on it (by method Copy), but the object
// itself will not be changed.
func (s *Storage) Iterate(
	ctx context.Context,
	tmpl data.Object,
	pref data.Prefix,
) (data.Iterator, error) {
	objects, err := s.Search(ctx, tmpl, pref)
	if err != nil {
		return nil, err
	}

	return data.NewSliceIterator(objects...), nil
}

func (s *Storage) path(t data.Type, k data.Key) string {
	return path.Join(string(t), string(k))
}

func (s *Storage) pref(t data.Type, p data.Prefix) string {
	return path.Join(string(t), string(p))
}
