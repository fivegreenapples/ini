package ini

import (
	"bytes"
	"testing"
)

func Test_AllSupportedFeatures(t *testing.T) {

	ini := `
empty = 
a = b
b = c
c[] = 123
c[] = 456
d[abc] = bob
d[def] = ajob
d[ghi] = "nob lob tob"

[sectionA]
a = b

`
	buf := bytes.NewBuffer([]byte(ini))
	res, err := newCanonicalFromReader(buf)
	t.Log(res)

	if err != nil {
		t.Error("unexpected error:", err)
	}

}
