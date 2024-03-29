package assets

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWei(t *testing.T) {
	for _, tt := range []struct {
		input string
		exp   string
	}{
		{"0", "0"},
		{"0 wei", "0"},
		{"0 ether", "0"},
		{"1", "1 wei"},
		{"1000", "1 kwei"},
		{"1100", "1.1 kwei"},
		{"1.1 kwei", "1.1 kwei"},
		{"1.1000 kwei", "1.1 kwei"},
		{"10. kwei", "10 kwei"},
		{"10.0 kwei", "10 kwei"},
		{"10.1 kwei", "10.1 kwei"},
		{"999.9 kwei", "999.9 kwei"},
		{"1000000", "1 mwei"},
		{"1000000000", "1 gwei"},
		{"1000000000000", "1 micro"},
		{"200000000000000", "200 micro"},
		{"200 micro", "200 micro"},
		{"0.2 milli", "200 micro"},
		{"281 micro", "281 micro"},
		{"281.474976710655 micro", "281.474976710655 micro"},
		{"0.281474976710655 milli", "281.474976710655 micro"},
		{"999.9 micro", "999.9 micro"},
		{"1000000000000000", "1 milli"},
		{"1000000000000000000", "1 ether"},
		{"1000000000000000000000", "1 kether"},
		{"1000000000000000000000000", "1 mether"},
		{"1000000000000000000000000000", "1 gether"},
		{"1000000000000000000000000000000", "1 tether"},
		{"1100000000000000000000000000000", "1.1 tether"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			var w Wei
			err := w.UnmarshalText([]byte(tt.input))
			require.NoError(t, err)
			b, err := w.MarshalText()
			require.NoError(t, err)
			assert.Equal(t, tt.exp, string(b))
			assert.Equal(t, tt.exp, w.String())
		})
	}
}

func FuzzWei(f *testing.F) {
	f.Add("1")
	f.Add("2.3 gwei")
	f.Add("00005 wei")
	f.Add("1100000000000000000000000000000")
	f.Add("1 wei")
	f.Add("2.3 kwei")
	f.Add("0.0005gwei")
	f.Add("1100000000000000000000000000000 wei")
	f.Add("9.7 tether")
	f.Add("0.567gether")
	f.Add("5.753 mether")
	f.Add("42 kether")
	f.Add("1 ether")
	f.Add("10.4 milli")
	f.Add("5 micro")
	f.Fuzz(func(t *testing.T, v string) {
		if len(v) > 1_000 {
			t.Skipf("too many characters: %d", len(v))
		}
		if e := tryParseExp(v); -1000 > e || e > 1000 {
			t.Skipf("exponent too large: %d", e)
		}
		var w Wei
		err := w.UnmarshalText([]byte(v))
		if err != nil {
			t.Skip()
		}

		b, err := w.MarshalText()
		require.NoErrorf(t, err, "failed to marshal %v after unmarshaling from %q", w, v)

		var w2 Wei
		err = w2.UnmarshalText(b)
		require.NoErrorf(t, err, "failed to unmarshal %s after marshaling from %v", string(b), w)
		require.Equal(t, w, w2, "unequal values after marshal/unmarshal")
	})
}

func tryParseExp(v string) int64 {
	i := strings.IndexAny(v, "Ee")
	if i == -1 {
		return -1
	}
	v = v[i+1:]
	if i := strings.IndexFunc(v, func(r rune) bool {
		switch {
		case r == '-' || r == '+':
			return false
		case r < '0' || '9' < r:
			return true
		}
		return false
	}); i > -1 {
		v = v[:i]
	}
	e, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return -1
	}
	return e
}

func Test_tryParseExp(t *testing.T) {
	got := tryParseExp("000000000E0000000060000000wei")
	assert.Equal(t, int64(60000000), got)
	got = tryParseExp("0e-80000800")
	assert.Equal(t, int64(-80000800), got)
	got = tryParseExp("0e+802444440")
	assert.Equal(t, int64(802444440), got)
}
