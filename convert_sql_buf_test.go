package freetds

import (
	"fmt"
	"github.com/stretchrcom/testify/assert"
	"testing"
	"time"
)

func TestInt(t *testing.T) {
	testToSqlToType(t, SYBINT4, 2147483647)
	testToSqlToType(t, SYBINT4, -2147483648)

	testToSqlToType(t, SYBINT4, int(2147483647))
	testToSqlToType(t, SYBINT4, int32(2147483647))
	testToSqlToType(t, SYBINT4, int64(2147483647))

	_, err := typeToSqlBuf(SYBINT4, "pero")
	assert.NotNil(t, err)
}

func TestInt16(t *testing.T) {
	testToSqlToType(t, SYBINT2, int16(32767))
	testToSqlToType(t, SYBINT2, int16(-32768))
	testToSqlToType(t, SYBINT2, 123)
	//overflow
	data, err := typeToSqlBuf(SYBINT2, 32768)
	assert.Nil(t, err)
	i16 := sqlBufToType(SYBINT2, data)
	assert.Equal(t, i16, -32768)
	//error
	_, err = typeToSqlBuf(SYBINT2, "pero")
	assert.NotNil(t, err)
}

func TestInt8(t *testing.T) {
	testToSqlToType(t, SYBINT1, uint8(127))
	testToSqlToType(t, SYBINT1, uint8(255))
	data, err := typeToSqlBuf(SYBINT1, 127)
	assert.Nil(t, err)
	value, _ := sqlBufToType(SYBINT1, data).(uint8)
	assert.Equal(t, int(value), 127)
}

func TestInt64(t *testing.T) {
	testToSqlToType(t, SYBINT8, int64(-9223372036854775808))
	testToSqlToType(t, SYBINT8, int64(9223372036854775807))
}

func TestFloat(t *testing.T) {
	testToSqlToType(t, SYBFLT8, float64(123.45))
	testToSqlToType(t, SYBFLT8, float32(123.5))
	testToSqlToType(t, SYBREAL, float32(123.45))
	testToSqlToType(t, SYBREAL, float64(123.5))
}

func TestBool(t *testing.T) {
	testToSqlToType(t, SYBBIT, false)
	testToSqlToType(t, SYBBIT, true)
}

func TestMoney(t *testing.T) {
	testToSqlToType(t, SYBMONEY4, float64(1223.45))
	testToSqlToType(t, SYBMONEY, float64(1223.45))
	testToSqlToType(t, SYBMONEY, float64(1234.56))
	testToSqlToType(t, SYBMONEY, float64(1234.56))
}

func TestTime(t *testing.T) {
	value := time.Now()
	typ := SYBDATETIME
	data, err := typeToSqlBuf(typ, value)
	assert.Nil(t, err)
	value2 := sqlBufToType(typ, data)
	value2t, _ := value2.(time.Time)
	diff := value2t.Sub(value)
	if diff > 3000000 && diff < -3000000 {
		t.Error()
		fmt.Printf("TestTime\n%s\n%s\ndiff: %d", value, value2t, diff)
	}
}

func TestTime4(t *testing.T) {
	value := time.Date(2014, 1, 5, 23, 24, 0, 0, time.UTC)
	testToSqlToType(t, SYBDATETIME4, value)
}

func TestBinary(t *testing.T) {
	value := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	testToSqlToType(t, SYBVARBINARY, value)
}

func testToSqlToType(t *testing.T, typ int, value interface{}) {
	data, err := typeToSqlBuf(typ, value)
	assert.Nil(t, err)
	value2 := sqlBufToType(typ, data)
	assert.Equal(t, value, value2)
}
