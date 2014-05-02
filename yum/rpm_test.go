package yum

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func rpmString(rpm RPM) string {
	return fmt.Sprintf("%s.%s-%d", rpm.Name(), rpm.Version(), rpm.Release())
}

func TestProvidesComparison(t *testing.T) {
	const name = "TestPackage"

	for _, table := range []struct {
		name     string
		input    RPMSlice
		expected []int
	}{
		{
			name: "Comparison",
			input: RPMSlice{
				NewProvides(name, "1.0.1", 2, 0, "EQ", nil),
				NewProvides(name, "1.0.1", 1, 0, "EQ", nil),
				NewProvides(name, "1.0.0", 1, 0, "EQ", nil),
			},
			expected: []int{2, 1, 0},
		},
		{
			name: "ComparisonNoRelease",
			input: RPMSlice{
				NewProvides(name, "1.0.1", 0, 0, "EQ", nil),
				NewProvides(name, "1.0.1", 1, 0, "EQ", nil),
				NewProvides(name, "1.0.0", 1, 0, "EQ", nil),
			},
			expected: []int{2, 0, 1},
		},
		{
			name: "ComparisonAlpha",
			input: RPMSlice{
				NewProvides(name, "1.0.9.B", 2, 0, "EQ", nil),
				NewProvides(name, "1.0.9.A", 1, 0, "EQ", nil),
				NewProvides(name, "1.0.0", 1, 0, "EQ", nil),
				NewProvides(name, "1.0.10.A", 1, 0, "EQ", nil),
			},
			expected: []int{2, 1, 0, 3},
		},
	} {
		exp := make(RPMSlice, 0, len(table.input))
		for _, idx := range table.expected {
			exp = append(exp, table.input[idx])
		}
		sort.Sort(table.input)

		if !reflect.DeepEqual(exp, table.input) {
			exp_str := "["
			for _, v := range exp {
				exp_str += rpmString(v)
			}
			exp_str += "]"
			in_str := "["
			for _, v := range table.input {
				in_str += rpmString(v)
			}
			in_str += "]"

			t.Fatalf("slices differ (%s):\nexp=%v\ngot=%v\n", table.name, exp_str, in_str)
		}
	}
}

// EOF
