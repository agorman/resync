package resync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVars(t *testing.T) {
	s := "hello"
	strp := String(s)
	assert.NotNil(t, strp)
	assert.Equal(t, *strp, s)
	assert.Equal(t, StringValue(strp), s)
	assert.Equal(t, StringValue(nil), "")

	i := 123
	intp := Int(i)
	assert.NotNil(t, intp)
	assert.Equal(t, *intp, i)
	assert.Equal(t, IntValue(intp), i)
	assert.Equal(t, IntValue(nil), 0)

	b := true
	boolp := Bool(b)
	assert.NotNil(t, boolp)
	assert.Equal(t, *boolp, b)
	assert.Equal(t, BoolValue(boolp), b)
	assert.Equal(t, BoolValue(nil), false)
}
