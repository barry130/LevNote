package enginescene

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestSceneNumericSyntaxAndExactCounts(t *testing.T) {
	for _, value := range []string{"1", "1.0", "1e0", "0.1e1", "100e-2", "1.000000000000000000000e+0000000", "100000000000000000000000000000e-29"} {
		t.Run(value, func(t *testing.T) {
			wire := strings.Replace(literalScene, `"schema":1`, `"schema":`+value, 1)
			wire = strings.Replace(wire, `"nodeCount":1`, `"nodeCount":`+value, 1)
			got, err := Decode([]byte(wire))
			if err != nil || got.Schema != 1 || len(got.Nodes) != 1 {
				t.Fatalf("exact integral spelling %q: %v", value, err)
			}
		})
	}
	for _, value := range []string{"1.00000000000000001", "0.99999999999999999999", "1e-1", "-1", "-0.1", "100000.00000000000001"} {
		requireDecodeError(t, []byte(strings.Replace(literalScene, `"nodeCount":1`, `"nodeCount":`+value, 1)), ErrInvalidScene)
	}
	for _, value := range []string{"01", "+1", ".1", "1.", "1e", "1e+", "1e--2", "0x1", "1_0", "NaN", "Infinity", "--1", "true", `"1"`} {
		requireDecodeError(t, []byte(strings.Replace(literalScene, `"x":-24`, `"x":`+value, 1)), ErrInvalidScene)
	}
	for _, tc := range []struct {
		text        string
		limit, want int
		err         error
	}{
		{"1e5", MaxNodes, MaxNodes, nil}, {"99999.0", MaxEdges, MaxEdges, nil},
		{"100000.0", MaxEdges, 0, ErrLimitExceeded}, {"100001", MaxNodes, 0, ErrLimitExceeded},
		{"-0e999999999999999", MaxEdges, 0, nil}, {"1e999999999999999999999", MaxNodes, 0, ErrLimitExceeded},
		{"100000000000000000000000000000e-29", MaxNodes, 1, nil},
	} {
		reader := sceneReader{data: []byte(tc.text)}
		got, err := reader.count(tc.limit)
		if got != tc.want || !errors.Is(err, tc.err) || reader.pos != len(tc.text) {
			t.Fatalf("count %q => %d/%v pos%d", tc.text, got, err, reader.pos)
		}
	}
}

func TestSceneIEEEConversionIndependentValues(t *testing.T) {
	const midpoint = "1.00000000000000011102230246251565404236316680908203125"
	// These decimal midpoints and expected IEEE bits are authored values. The
	// very long tails distinguish sticky rounding without a full-token parser.
	above := midpoint + strings.Repeat("0", 900) + "1"
	below := strings.TrimSuffix(midpoint, "5") + "4" + strings.Repeat("9", 900)
	cases := []struct {
		name, text string
		bits       uint64
	}{
		{"ordinary", "-1.25e2", math.Float64bits(-125)},
		{"fraction", "125e-3", math.Float64bits(0.125)},
		{"min-subnormal", "4.9406564584124654e-324", 1},
		{"negative-min-subnormal", "-4.9406564584124654e-324", 1<<63 | 1},
		{"subnormal-round-up", "3e-324", 1},
		{"subnormal-round-down", "2e-324", 0},
		{"negative-zero", "-0e99999999999999999", 1 << 63},
		{"underflow", "1e-9999999999999999999", 0},
		{"negative-underflow", "-1e-9999999999999999999", 1 << 63},
		{"long-coefficient-canceled", "1" + strings.Repeat("0", 100_000) + "e-100000", math.Float64bits(1)},
		{"long-fraction-canceled", "0." + strings.Repeat("0", 10_000) + "125e10001", math.Float64bits(1.25)},
		{"exact-midpoint-even", midpoint, math.Float64bits(1)},
		{"above-midpoint-sticky", above, math.Float64bits(1) + 1},
		{"below-midpoint-sticky", below, math.Float64bits(1)},
		{"max-coordinate", "1000000000.00000", math.Float64bits(MaxCoordinateAbs)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := sceneReader{data: []byte(tc.text)}
			value, err := reader.coordinate(false)
			if err != nil || math.Float64bits(value) != tc.bits || reader.pos != len(tc.text) {
				t.Fatalf("IEEE bits=%016x want=%016x err=%v pos=%d/%d", math.Float64bits(value), tc.bits, err, reader.pos, len(tc.text))
			}
			wire := strings.Replace(literalScene, `"x":-24`, `"x":`+tc.text, 1)
			decoded, err := Decode([]byte(wire))
			if err != nil || math.Float64bits(decoded.Bounds.X) != tc.bits {
				t.Fatalf("public IEEE bits=%016x err=%v", math.Float64bits(decoded.Bounds.X), err)
			}
		})
	}
}

func TestSceneNumericBoundsAndSaturatedExponent(t *testing.T) {
	for _, tc := range []struct {
		value     string
		dimension bool
		want      error
	}{
		{"1000000000.00000000000000000000001", false, ErrLimitExceeded},
		{"-1000000000.00000000000000000000001", false, ErrLimitExceeded},
		{"1.00000000000000001e9", false, ErrLimitExceeded},
		{"10000000.000000000000000001", true, ErrLimitExceeded},
		{"0", true, ErrInvalidScene}, {"-0", true, ErrInvalidScene}, {"-1", true, ErrInvalidScene},
		{"1e-100000", true, ErrInvalidScene}, {"1e999999999999999999999", false, ErrLimitExceeded},
	} {
		reader := sceneReader{data: []byte(tc.value)}
		_, err := reader.coordinate(tc.dimension)
		if !errors.Is(err, tc.want) {
			t.Fatalf("coordinate %q dimension=%v err=%v", tc.value, tc.dimension, err)
		}
		field := `"x":-24`
		if tc.dimension {
			field = `"width":196`
		}
		colon := strings.IndexByte(field, ':')
		requireDecodeError(t, []byte(strings.Replace(literalScene, field, field[:colon+1]+tc.value, 1)), tc.want)
	}
	for _, value := range []string{"1e7", "4.9406564584124654e-324"} {
		reader := sceneReader{data: []byte(value)}
		got, err := reader.coordinate(true)
		if err != nil || got <= 0 || got > MaxDimension {
			t.Fatalf("valid positive dimension %q => %g %v", value, got, err)
		}
	}
	// No big.Int/Rat or whole-token allocation is involved. Even a long
	// exponent with a late small value remains a standard accepted spelling.
	long := "1e" + strings.Repeat("0", 100_000) + "0"
	reader := sceneReader{data: []byte(long)}
	number, err := reader.number()
	if err != nil || number.exponent != 0 || number.kept != 1 || reader.pos != len(long) {
		t.Fatalf("long zero exponent: kept%d exponent%d err=%v", number.kept, number.exponent, err)
	}
	long = "1e" + strings.Repeat("9", 100_000)
	reader = sceneReader{data: []byte(long)}
	number, err = reader.number()
	if err != nil || number.exponent != MaxPayloadBytes+1024 || number.kept != 1 || reader.pos != len(long) {
		t.Fatalf("saturated exponent: kept%d exponent%d err=%v", number.kept, number.exponent, err)
	}
	_, err = number.integer(MaxNodes)
	if !errors.Is(err, ErrLimitExceeded) {
		t.Fatal(err)
	}
}
