package geometry

import "testing"

func TestParseBBoxValid(t *testing.T) {
	bbox, err := ParseBBox("-70.8,-33.6,-70.4,-33.3")
	if err != nil {
		t.Fatalf("ParseBBox returned error: %v", err)
	}
	want := BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	if bbox != want {
		t.Fatalf("ParseBBox = %+v, want %+v", bbox, want)
	}
	if got := bbox.String(); got != "-70.8,-33.6,-70.4,-33.3" {
		t.Fatalf("String() = %q", got)
	}
	if array := bbox.Array(); array != [4]float64{-70.8, -33.6, -70.4, -33.3} {
		t.Fatalf("Array() = %v", array)
	}
}

func TestParseBBoxInvalid(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"too few values", "-70.8,-33.6,-70.4"},
		{"too many values", "-70.8,-33.6,-70.4,-33.3,0"},
		{"not a number", "-70.8,west,-70.4,-33.3"},
		{"longitude out of range", "-200,-33.6,-70.4,-33.3"},
		{"latitude out of range", "-70.8,-33.6,-70.4,200"},
		{"min lon not less than max", "-70.4,-33.6,-70.8,-33.3"},
		{"min lat not less than max", "-70.8,-33.3,-70.4,-33.6"},
		{"empty", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseBBox(tc.input); err == nil {
				t.Fatalf("ParseBBox(%q) = nil error, want error", tc.input)
			}
		})
	}
}
