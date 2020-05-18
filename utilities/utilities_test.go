package utilities

import "testing"

func TestFind(t *testing.T) {
	inputArr := []string{"rohith", "uppala"}
	element := "rohith"
	got, isFound := Find(inputArr, element)
	wanted := 0
	if isFound == false || got != wanted {
		t.Fail()
	}
	element = "alex"
	got, isFound = Find(inputArr, "alex")
	wanted = -1
	if isFound == true || got != wanted {
		t.Fail()
	}
}
