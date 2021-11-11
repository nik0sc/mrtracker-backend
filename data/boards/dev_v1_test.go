package boards

import (
	"reflect"
	"testing"
)

func Test_DevV1(t *testing.T) {
	counter := make(map[TM1638OutSpec]int)

	for k, v := range DevV1 {
		for i, vv := range v {
			if vv.Chip > 2 {
				t.Errorf("Chip out of bounds: %s[%d]=%+v", k, i, vv)
			}

			if vv.Grid == 0 || vv.Grid > 8 {
				t.Errorf("Grid out of bounds: %s[%d]=%+v", k, i, vv)
			}

			if vv.Seg == 0 || vv.Seg > 10 {
				t.Errorf("Seg out of bounds: %s[%d]=%+v", k, i, vv)
			}

			counter[vv] += 1
		}
	}

	if len(counter) != 240 {
		t.Errorf("counter length: expected 240, got %d", len(counter))
	}

	for k, cnt := range counter {
		if cnt != 1 {
			t.Errorf("TM1638OutSpec=%+v: count %d", k, cnt)
		}
	}

	typ := reflect.TypeOf(TM1638OutSpec{})
	t.Logf("member size=%d align=%d fieldalign=%d", typ.Size(), typ.Align(), typ.FieldAlign())
}
