package buffer_test

import (
	"reflect"
	"testing"

	"github.com/genshinsim/gcsim/pkg/agg/util"
)

const SAMPLE_RATE = 4096

func TestIntFormat(t *testing.T) {
	b := util.NewIntBuffer(SAMPLE_RATE)

	// Test that it is actually an Int buffer
	if b.DataFormat() != util.FMT_INT16 {
		t.Errorf("IntBuffer not identifying as %s\n", util.FMT_INT16)
	}

	if _, ok := b.Frame().([]int16); !ok {
		t.Errorf("IntBuffer raw data tyoe is not %s, is actually %s\n", util.FMT_INT16, reflect.TypeOf(b.Frame()))
	}
}

func TestIntData(t *testing.T) {
	b := util.NewIntBuffer(SAMPLE_RATE)
	println(b)

}
