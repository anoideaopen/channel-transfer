package data

// ToType is an helper function with which you can wrap a call to the interator
// Next() and get an object instance, without an additional type conversion. The
// function will not compile in go versions earlier than 1.18. An example is in
// the file convert_test.go.
func ToType[T any](in Object, err error) (*T, error) {
	if err != nil {
		return nil, err
	}

	var iface interface{} = in

	if obj, ok := iface.(*T); ok {
		return obj, nil
	}

	return nil, ErrInvalidType
}

// ToSlice is an helper function with which you can wrap a call to Storage
// Search() and get a slice of the identical objects, without additional type
// conversion. The function will not compile in go versions smaller than 1.18.
// An example is in the file convert_test.go.
func ToSlice[T any](in []Object, err error) ([]*T, error) {
	out := make([]*T, 0, len(in))
	for _, item := range in {
		obj, err := ToType[T](item, err)
		if err != nil {
			return nil, err
		}

		out = append(out, obj)
	}

	return out, nil
}
