package redis

import (
	"testing"
)

func TestSimpleGetAndSet(t *testing.T) {
	conn := CreateConnection(0)
	err := conn.SetKey("foo", "bar")
	if err != nil {
		t.Fatalf("Error during redis set")
	}
	got, _ := conn.GetKey("foo")
	expected := "bar"
	if expected != got {
		t.Fatalf("Simple get and set Failed")
	}
}
