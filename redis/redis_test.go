package redis

import "testing"

func testSimpleGetAndSet(t *testing.T) {
	redisConn := CreateConnection()
	err := redisConn.set("foo", "bar")
	if err != nil {
		t.Fatalf("Error during redis set")
	}
	got := redisConn.get("foo")
	expected := "bar"
	if expected != got {
		t.Fatalf("Simple get and set Failed")
	}
}
