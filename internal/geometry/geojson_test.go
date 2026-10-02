package geometry

import "testing"

func TestBBoxFromGeoJSONPolygon(t *testing.T) {
	data := []byte(`{
		"type": "Polygon",
		"coordinates": [[[-70.72,-33.72],[-70.68,-33.72],[-70.68,-33.69],[-70.72,-33.69],[-70.72,-33.72]]]
	}`)
	bbox, err := BBoxFromGeoJSON(data)
	if err != nil {
		t.Fatalf("BBoxFromGeoJSON returned error: %v", err)
	}
	want := BBox{MinLon: -70.72, MinLat: -33.72, MaxLon: -70.68, MaxLat: -33.69}
	if bbox != want {
		t.Fatalf("bbox = %+v, want %+v", bbox, want)
	}
}

func TestBBoxFromGeoJSONMultiPolygon(t *testing.T) {
	data := []byte(`{
		"type": "MultiPolygon",
		"coordinates": [
			[[[0,0],[1,0],[1,1],[0,1],[0,0]]],
			[[[2,2],[4,2],[4,4],[2,4],[2,2]]]
		]
	}`)
	bbox, err := BBoxFromGeoJSON(data)
	if err != nil {
		t.Fatalf("BBoxFromGeoJSON returned error: %v", err)
	}
	want := BBox{MinLon: 0, MinLat: 0, MaxLon: 4, MaxLat: 4}
	if bbox != want {
		t.Fatalf("bbox = %+v, want %+v", bbox, want)
	}
}

func TestBBoxFromGeoJSONFeature(t *testing.T) {
	data := []byte(`{
		"type": "Feature",
		"properties": {},
		"geometry": {"type": "Polygon", "coordinates": [[[10,20],[11,20],[11,21],[10,21],[10,20]]]}
	}`)
	bbox, err := BBoxFromGeoJSON(data)
	if err != nil {
		t.Fatalf("BBoxFromGeoJSON returned error: %v", err)
	}
	want := BBox{MinLon: 10, MinLat: 20, MaxLon: 11, MaxLat: 21}
	if bbox != want {
		t.Fatalf("bbox = %+v, want %+v", bbox, want)
	}
}

func TestBBoxFromGeoJSONFeatureCollection(t *testing.T) {
	data := []byte(`{
		"type": "FeatureCollection",
		"features": [
			{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,1],[0,0]]]}},
			{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[5,5],[7,5],[7,7],[5,7],[5,5]]]}}
		]
	}`)
	bbox, err := BBoxFromGeoJSON(data)
	if err != nil {
		t.Fatalf("BBoxFromGeoJSON returned error: %v", err)
	}
	want := BBox{MinLon: 0, MinLat: 0, MaxLon: 7, MaxLat: 7}
	if bbox != want {
		t.Fatalf("bbox = %+v, want %+v", bbox, want)
	}
}

func TestBBoxFromGeoJSONInvalid(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{"invalid json", `{`},
		{"missing type", `{}`},
		{"unsupported type", `{"type":"Point","coordinates":[0,0]}`},
		{"unsupported geometry", `{"type":"Feature","geometry":{"type":"Point","coordinates":[0,0]}}`},
		{"empty feature collection", `{"type":"FeatureCollection","features":[]}`},
		{"null geometry", `{"type":"Feature","geometry":null}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BBoxFromGeoJSON([]byte(tc.data)); err == nil {
				t.Fatalf("BBoxFromGeoJSON(%s) = nil error, want error", tc.data)
			}
		})
	}
}
