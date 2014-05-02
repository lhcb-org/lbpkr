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

func TestFlags(t *testing.T) {
	const name = "TestPackage"
	const v1 = "1.0.1"
	const v2 = "1.2.0"
	const v3 = "1.3.5"
	const rel = 2

	// checking simple comparison
	ps := RPMSlice{
		NewProvides(name, v1, rel, 0, "EQ", nil),
		NewProvides(name, v2, rel, 0, "EQ", nil),
		NewProvides(name, v3, rel, 0, "EQ", nil),
	}

	for _, table := range []struct {
		ctor string
		vals []bool
		strs []string
	}{
		{
			ctor: "GT",
			vals: []bool{false, false, true},
			strs: []string{
				"%s not %s %s",
				"%s not %s %s",
				"%s %s %s",
			},
		},
		{
			ctor: "GE",
			vals: []bool{false, true, true},
			strs: []string{
				"%s not %s %s",
				"%s %s %s",
				"%s %s %s",
			},
		},
		{
			ctor: "LT",
			vals: []bool{true, false, false},
			strs: []string{
				"%s %s %s",
				"%s not %s %s",
				"%s not %s %s",
			},
		},
		{
			ctor: "LE",
			vals: []bool{true, true, false},
			strs: []string{
				"%s %s %s",
				"%s %s %s",
				"%s not %s %s",
			},
		},
	} {
		req := NewRequires(name, v2, rel, 0, table.ctor, "")
		for i := range table.vals {
			o := req.ProvideMatches(ps[i])
			if o != table.vals[i] {
				t.Fatalf(table.strs[i], ps[i], table.ctor, req)
			}
		}
	}

}

func TestRequiresWithNoVersion(t *testing.T) {
	const name = "TestPackage"
	const v1 = "1.0.1"
	const rel = 2

	// checking simple comparison
	p1 := NewProvides(name, v1, rel, 0, "EQ", nil)
	req := NewRequires(name, "", 0, 0, "EQ", "")
	if !req.ProvideMatches(p1) {
		t.Fatalf("expected %s to provide for %s\n", p1, req)
	}
}

func TestRequiresDifferentName(t *testing.T) {
	const name = "TestPackage"
	const v1 = "1.0.1"
	const rel = 2

	// checking simple comparison
	p1 := NewProvides(name, v1, rel, 0, "EQ", nil)
	req := NewRequires(name+"XYZ", "", 0, 0, "EQ", "")
	if req.ProvideMatches(p1) {
		t.Fatalf("expected %s to NOT provide for %s\n", p1, req)
	}
}

func TestOrderWithDifferentName(t *testing.T) {
	const name = "TestPackage"
	const v1 = "1.0.1"
	const rel = 2

	// checking simple comparison
	p1 := NewProvides(name, v1, rel, 0, "EQ", nil)
	p2 := NewProvides(name+"z", v1, rel, 0, "EQ", nil)
	if !RpmLessThan(p1, p2) {
		t.Fatalf("expected %s < %s\n", p1, p2)
	}
}

// EOF
