package ini

import (
	"testing"
)

type scalars struct {
	Bool   bool
	Int    int
	String string
	Float  float64
	Slice  []int
	Array  [5]string
	Map    map[string]int
}

type sectioned struct {
	scalars
	Global     int
	SectionOne scalars
	SectionTwo scalars
}

func Test_Struct(t *testing.T) {

	input := sectioned{
		scalars: scalars{
			String: "anon",
		},
		Global: 99,
		SectionOne: scalars{
			Bool:   true,
			Int:    1,
			String: "one space two",
			Float:  1.1,
			Slice:  []int{6, 4, 2},
			Array:  [5]string{"A", "r", "r", "a", "y"},
			Map:    map[string]int{"a": 1, "b": 2},
		},
	}

	ini, err := Marshal(&input)
	if err != nil {
		t.Fatalf("unexpected error from marshalling basic struct: %v", err)
	}

	t.Log(string(ini))
}
