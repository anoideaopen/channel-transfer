package redis

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/redis/go-redis/v9"
)

// Compatibility check at compile time.
var _ data.Object = &TestObject{}

type TestObject struct {
	Field1 string
	Field2 int
}

func (to *TestObject) MarshalBinary() (data []byte, err error) {
	return json.Marshal(to)
}

func (to *TestObject) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, to)
}

func (to *TestObject) Clone() data.Object {
	return &TestObject{
		Field1: to.Field1,
		Field2: to.Field2,
	}
}

func (to *TestObject) Instance() data.Type {
	return data.InstanceOf(to)
}

func (to *TestObject) Equal(other *TestObject) bool {
	return to.Field1 == other.Field1 && to.Field2 == other.Field2
}

func TestInMemRedisStorage(t *testing.T) {
	objects := []*TestObject{
		{Field1: "Hello", Field2: 42},
		{Field1: "World", Field2: 24},
		{Field1: "Hola Amigo", Field2: 12},
	}

	mredis := miniredis.RunT(t)

	repo, err := NewStorage(
		context.Background(),
		redis.NewUniversalClient(
			&redis.UniversalOptions{
				Addrs: []string{mredis.Addr()},
			},
		),
		time.Hour,
		"test",
	)
	if err != nil {
		t.Fatalf("failed to save object: %v", err)
	}

	if err := repo.Save(context.TODO(), objects[0], "00"); err != nil {
		t.Fatalf("failed to save object: %v", err)
	}

	if err := repo.Save(context.TODO(), objects[1], "01"); err != nil {
		t.Fatalf("failed to save object: %v", err)
	}

	if err := repo.Save(context.TODO(), objects[2], "11"); err != nil {
		t.Fatalf("failed to save object: %v", err)
	}

	var obj0, obj1, obj2 TestObject

	if err := repo.Load(context.TODO(), &obj0, "00"); err != nil {
		t.Fatalf("failed to load object 0: %v", err)
	}

	if err := repo.Load(context.TODO(), &obj1, "01"); err != nil {
		t.Fatalf("failed to load object 1: %v", err)
	}

	if err := repo.Load(context.TODO(), &obj2, "32"); !errors.Is(
		err,
		data.ErrObjectNotFound,
	) {
		t.Fatalf("incorrect error: %v", err)
	}

	if !obj0.Equal(objects[0]) {
		t.Fatalf("object 0 is not equal to obj0: %+v", obj0)
	}

	if !obj1.Equal(objects[1]) {
		t.Fatalf("object 1 is not equal to obj1: %+v", obj1)
	}

	collection, err := data.ToSlice[TestObject](
		repo.Search(context.TODO(), &TestObject{}, ""),
	)
	if err != nil {
		t.Fatalf("failed search objects: %v", err)
	}

	if len(collection) != len(objects) {
		t.Fatalf("invalid objects count: %d", len(collection))
	}

	collection, err = data.ToSlice[TestObject](
		repo.Search(context.TODO(), &TestObject{}, "0"),
	)
	if err != nil {
		t.Fatalf("failed search objects: %v", err)
	}

	if len(collection) != len(objects)-1 {
		t.Fatalf("invalid objects count: %d", len(collection))
	}

	_, err = repo.Search(context.TODO(), &TestObject{}, "32")
	if !errors.Is(err, data.ErrObjectNotFound) {
		t.Fatal("incorrect search processing")
	}

	iter, err := repo.Iterate(context.TODO(), &TestObject{}, "")
	if err != nil {
		t.Fatalf("failed to create iterator: %v", err)
	}
	defer func() {
		_ = iter.Close()
	}()

	cnt := 0
	for iter.HasNext() {
		obj, err := data.ToType[TestObject](iter.Next())
		if err != nil {
			t.Fatalf("failed to load object: %v", err)
		}

		for _, object := range objects {
			if obj.Equal(object) {
				cnt++
				break
			}
		}
	}
	if cnt != len(objects) {
		t.Fatalf("objects is not equal : expected %d - actual %d", len(objects), cnt)
	}
}
