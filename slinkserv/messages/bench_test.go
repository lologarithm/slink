package messages

import (
	"testing"
	"time"
)

func BenchmarkUnmarshal(b *testing.B) {
	b.StopTimer()
	a := &A{
		Name:     "asdf",
		BirthDay: time.Now().Unix(),
		Phone:    "123-234-4567",
		Siblings: 2,
		Spouse:   1,
		Money:    134.345,
	}
	data := make([]byte, a.Len())
	a.Serialize(data)

	b.ReportAllocs()
	obj := &A{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		obj.Deserialize(data)
	}
}

func BenchmarkMarshal(b *testing.B) {
	b.StopTimer()
	a := &A{
		Name:     "asdf",
		BirthDay: time.Now().Unix(),
		Phone:    "123-234-4567",
		Siblings: 2,
		Spouse:   1,
		Money:    134.345,
	}
	data := make([]byte, a.Len())

	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		a.Serialize(data)
	}
}
