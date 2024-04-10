// Package data contains primitives for working with an object that stores data
// used in other parts of the library.
package data

import (
	"encoding"
	"reflect"
	"strings"
)

// Type is used as a unique name for a set of objects of the same type. It is
// intended to divide the namespace between the stored data. In general, a
// structure name obtained through the reflect package can be used, but when
// renaming a structure or refactoring code, the uniqueness of this value must
// be taken into account.
type Type string

// InstanceOf is a simple implementation of getting a unique type for the
// stored data. It works using the reflect package and returns the data type
// used as namespace.
func InstanceOf(v interface{}) Type {
	return Type(strings.ReplaceAll(reflect.TypeOf(v).String(), "*", "_"))
}

// Object is an interface which is designed to store and load data.
// This interface must be implemented by all objects that interact and store
// their state in the HAL. You can see an example in object_test.go file.
type Object interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	// Clone should create an exact copy of the object, located in a different
	// memory location from the original. This is necessary in case of cache
	// optimization to avoid marshalling the object in some cases. Clone is also
	// used as a template for finding an object in the repository.
	Clone() Object

	// Instance should return a unique object type to share namespace between
	// the stored data. In the simplest case, you can return the type name via
	// InstanceOf, but keep in mind that you need to preserve compatibility or
	// provide for migration when refactoring.
	Instance() Type
}
