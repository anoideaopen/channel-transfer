package data

import "context"

type (
	// Key is a unique name that is used to save and load an Object. At the same
	// time as Key, the object type also takes part in the saving.
	Key string

	// Prefix is the part of Key by which objects in the repository can be
	// searched and iterated.
	Prefix string
)

// Storage is a base interface that provides operations for storing, loading and
// retrieving objects of type Object in an abstract object repository. This
// storage should be implemented for testing, real-world application and further
// use in tokens.
type Storage interface {
	// Save stores the object in a storage, using Key as the unique name (path)
	// to save the object and Type (by calling Instance method) to share the
	// namespace between objects. Key can be repeated for different Type values.
	// The MarshalBinary method can be called before the object is saved.
	Save(context.Context, Object, Key) error

	// Load loads an object from the repository, using Key as a unique name
	// (path) to find the object and Type (by calling the Instance method). The
	// UnmarshalBinary method can be called for the object. To restore an
	// object, you must allocate memory for it to avoid panic when working with
	// a nil address.
	// If the object is not found, an ErrObjectNotFound error will be returned.
	Load(context.Context, Object, Key) error

	// Search searches for objects in the repository of a certain Type. It
	// returns a list of found objects or ErrObjectNotFound if no objects
	// matching the criteria were found. Prefix is used for additional filtering
	// of objects according to their Key with partial match at the beginning of
	// the key. If you want to get all objects Prefix can be empty. The original
	// object is only needed as a template, other instances will be created
	// based on it (by method Copy), but the object itself will not be changed.
	Search(context.Context, Object, Prefix) ([]Object, error)

	// Iterate does what Search does, but in place of the finished collection of
	// objects an iterator will be returned that can be used to list the objects
	// one by one. The original object is only needed as a template, other
	// instances will be created based on it (by method Copy), but the object
	// itself will not be changed.
	Iterate(context.Context, Object, Prefix) (Iterator, error)
}

// Iterator is an interface that allows you to cycle through the objects in the
// repository one by one. To prevent memory leaks, you should close the iterator
// every time you finish working with the data.
type Iterator interface {
	// HasNext returns true if it is possible to get the next object from the
	// collection.
	HasNext() bool

	// Next returns the next object from the iterator. This object is already
	// unmarshaled and can be converted using type casting.
	Next() (Object, error)

	// Close should be called every time the collection is finished. This method
	// frees the iterator data in memory.
	Close() error
}
