package redis

import (
	"context"
	"path"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/redis/go-redis/v9"
)

const (
	notFound        = "redis: nil"
	TTLNotTakenInto = time.Duration(0)
)

var _ data.Storage = &Storage{}

// Storage implements a data.Storage interface using a map stored in memory.
type Storage struct {
	rdb      redis.UniversalClient
	ttl      time.Duration
	dbPrefix string
}

// NewStorage creates an instance of the Storage structure with Redis client
// instance.
func NewStorage(
	rdb redis.UniversalClient,
	ttl time.Duration,
	dbPrefix string,
) (*Storage, error) {
	if err := rdb.Ping(context.TODO()).Err(); err != nil {
		return nil, err
	}

	return &Storage{
		rdb:      rdb,
		ttl:      ttl,
		dbPrefix: dbPrefix,
	}, nil
}

// Save stores the object in a storage, using Key as the unique name (path)
// to save the object and Type (by calling Instance method) to share the
// namespace between objects. Key can be repeated for different Type values.
// The MarshalBinary method can be called before the object is saved.
func (s *Storage) Save(
	ctx context.Context,
	obj data.Object,
	key data.Key,
) error {
	bin, err := obj.MarshalBinary()
	if err != nil {
		return err
	}

	return s.rdb.Set(
		ctx,
		s.path(obj.Instance(), key),
		bin,
		s.ttl,
	).Err()
}

// Load loads an object from the repository, using Key as a unique name
// (path) to find the object and Type (by calling the Instance method). The
// UnmarshalBinary method can be called for the object. To restore an
// object, you must allocate memory for it to avoid panic when working with
// a nil address.
// If the object is not found, an ErrObjectNotFound error will be returned.
func (s *Storage) Load(
	ctx context.Context,
	obj data.Object,
	key data.Key,
) error {
	bin, err := s.rdb.Get(ctx, s.path(obj.Instance(), key)).Bytes()
	if err != nil {
		if err.Error() == notFound {
			return data.ErrObjectNotFound
		}

		return err
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
	ctx context.Context,
	tmpl data.Object,
	pref data.Prefix,
) ([]data.Object, error) {
	iter := s.rdb.Scan(
		ctx,
		0,
		s.path(tmpl.Instance(), data.Key(pref))+"*",
		0,
	).Iterator()

	const defaultObjectCapacity = 255

	var (
		out = make([]data.Object, 0, defaultObjectCapacity)
		bin []byte
		err error
	)
	for iter.Next(ctx) {
		clone := tmpl.Clone()

		if bin, err = s.rdb.Get(ctx, iter.Val()).Bytes(); err != nil {
			if err.Error() == notFound {
				continue
			}

			return nil, err
		}

		if err := clone.UnmarshalBinary(bin); err != nil {
			return nil, err
		}

		out = append(out, clone)
	}

	if err := iter.Err(); err != nil {
		return nil, err
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
	return path.Join(s.dbPrefix, string(t), string(k))
}

func (s *Storage) SaveConsideringTTL(
	ctx context.Context,
	obj data.Object,
	key data.Key,
	ttl time.Duration,
) error {
	bin, err := obj.MarshalBinary()
	if err != nil {
		return err
	}

	if ttl == TTLNotTakenInto {
		_, err = s.rdb.Get(ctx, s.path(obj.Instance(), key)).Bytes()
		if err != nil {
			if err.Error() != notFound {
				return err
			}
			ttl = s.ttl
		} else {
			ttl = s.rdb.TTL(ctx, s.path(obj.Instance(), key)).Val()
		}
	}

	return s.rdb.Set(
		ctx,
		s.path(obj.Instance(), key),
		bin,
		ttl,
	).Err()
}

func (s *Storage) TTL() time.Duration {
	return s.ttl
}
