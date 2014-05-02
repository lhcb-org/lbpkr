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

func TestMatchEqual(t *testing.T) {
	const name = "TestPackage"
	const v1 = "1.0.1"
	const v2 = "1.2.0"
	const rel1 = 2
	const rel2 = 3

	// checking equality
	p1 := NewProvides(name, v1, rel1, 0, "EQ", nil)
	req := NewRequires(name, v1, rel1, 0, "EQ", "")
	if !req.ProvideMatches(p1) {
		t.Fatalf("%s should match %s.\n", rpmString(p1), rpmString(req))
	}

	// checking release mismatch
	p2 := NewProvides(name, v1, rel2, 0, "EQ", nil)
	if req.ProvideMatches(p2) {
		t.Fatalf("%s should NOT match %s.\n", rpmString(p2), rpmString(req))
	}

	// checking version mismatch
	p3 := NewProvides(name, v2, rel1, 0, "EQ", nil)
	if req.ProvideMatches(p3) {
		t.Fatalf("%s should NOT match %s.\n", rpmString(p3), rpmString(req))
	}

	// checking name mismatch
	p4 := NewProvides(name+"XYZ", v1, rel1, 0, "EQ", nil)
	if req.ProvideMatches(p4) {
		t.Fatalf("%s should NOT match %s.\n", rpmString(p4), rpmString(req))
	}

}

// EOF
