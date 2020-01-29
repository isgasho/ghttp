package ghttp

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testInvalidVal            = complex(1, 2)
	testStringVal             = "hi"
	testBytesVal              = []byte{'h', 'e', 'l', 'l', 'o'}
	testBytesValStr           = "hello"
	testBoolValStr            = "true"
	testIntVal                = -314
	testIntValStr             = "-314"
	testInt8Val       int8    = -128
	testInt8ValStr            = "-128"
	testInt16Val      int16   = -32768
	testInt16ValStr           = "-32768"
	testInt32Val      int32   = -314159
	testInt32ValStr           = "-314159"
	testInt64Val      int64   = -31415926535
	testInt64ValStr           = "-31415926535"
	testUintVal       uint    = 314
	testUintValStr            = "314"
	testUint8Val      uint8   = 127
	testUint8ValStr           = "127"
	testUint16Val     uint16  = 32767
	testUint16ValStr          = "32767"
	testUint32Val     uint32  = 314159
	testUint32ValStr          = "314159"
	testUint64Val     uint64  = 31415926535
	testUint64ValStr          = "31415926535"
	testFloat32Val    float32 = 3.14159
	testFloat32ValStr         = "3.14159"
	testFloat64Val            = 3.1415926535
	testFloat64ValStr         = "3.1415926535"
	testNumberVal             = Number(testFloat64Val)

	testStringSlice = []string{"hello", "world"}
	testBoolSlice   = []bool{true, false}
	testNumberSlice = []Number{-3.1415926535, 3.1415926535}

	testValues = Values{
		"stringVal":   testStringVal,
		"stringSlice": testStringSlice,
	}
	testHeaders = Headers{
		"string-val":   testStringVal,
		"string-slice": testStringSlice,
	}
	testJSON = map[string]interface{}{
		"stringVal":   testStringVal,
		"boolVal":     true,
		"numberVal":   testNumberVal,
		"stringSlice": testStringSlice,
		"boolSlice":   testBoolSlice,
		"numberSlice": testNumberSlice,
	}
)



// 用于测试 Client 是否复用连接
func printLocalDial(ctx context.Context, network, addr string) (net.Conn, error) {
	dial := net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	conn, err := dial.DialContext(ctx, network, addr)
	if err != nil {
		return conn, err
	}

	fmt.Printf("network connected at %s\n", conn.LocalAddr().String())
	return conn, err
}

func TestNumber(t *testing.T) {
	assert.Equal(t, int(testNumberVal), testNumberVal.Int())
	assert.Equal(t, int8(testNumberVal), testNumberVal.Int8())
	assert.Equal(t, int16(testNumberVal), testNumberVal.Int16())
	assert.Equal(t, int32(testNumberVal), testNumberVal.Int32())
	assert.Equal(t, int64(testNumberVal), testNumberVal.Int64())
	assert.Equal(t, uint(testNumberVal), testNumberVal.Uint())
	assert.Equal(t, uint8(testNumberVal), testNumberVal.Uint8())
	assert.Equal(t, uint16(testNumberVal), testNumberVal.Uint16())
	assert.Equal(t, uint32(testNumberVal), testNumberVal.Uint32())
	assert.Equal(t, uint64(testNumberVal), testNumberVal.Uint64())
	assert.Equal(t, float32(testNumberVal), testNumberVal.Float32())
	assert.Equal(t, float64(testNumberVal), testNumberVal.Float64())
	assert.Equal(t, testFloat64ValStr, testNumberVal.String())
}

func TestH(t *testing.T) {
	var h H
	enc, err := json.Marshal(testJSON)
	require.NoError(t, err)
	err = json.Unmarshal(enc, &h)
	if assert.NoError(t, err) {
		assert.True(t, h.Has("stringVal"))
		assert.Equal(t, testStringVal, h.Get("stringVal"))
		assert.True(t, h.GetBoolDefault("noKey", true))
		assert.True(t, h.GetBool("boolVal"))
		assert.Equal(t, testBoolSlice, h.GetBoolSlice("boolSlice"))
		assert.Equal(t, testStringVal, h.GetStringDefault("noKey", testStringVal))
		assert.Equal(t, testStringVal, h.GetString("stringVal"))
		assert.Equal(t, testStringSlice, h.GetStringSlice("stringSlice"))
		assert.Equal(t, testNumberVal, h.GetNumberDefault("noKey", testNumberVal))
		assert.Equal(t, testNumberVal, h.GetNumber("numberVal"))
		assert.Equal(t, testNumberSlice, h.GetNumberSlice("numberSlice"))
	}

	h = H{
		"msg": "hello world",
	}
	want := "{\n\t\"msg\": \"hello world\"\n}"
	assert.Equal(t, want, h.String())
}

func TestH_Decode(t *testing.T) {
	type user struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	}
	h := H{
		"code": 200,
		"user": map[string]interface{}{
			"id":   10086,
			"name": "ghttp",
		},
	}
	u := new(user)
	err := h.GetH("user").Decode(u)
	if assert.NoError(t, err) {
		assert.Equal(t, 10086, u.Id)
		assert.Equal(t, "ghttp", u.Name)
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		input interface{}
		want  string
	}{
		{testStringVal, testStringVal},
		{testBytesVal, testBytesValStr},
		{true, testBoolValStr},
		{testIntVal, testIntValStr},
		{testInt8Val, testInt8ValStr},
		{testInt16Val, testInt16ValStr},
		{testInt32Val, testInt32ValStr},
		{testInt64Val, testInt64ValStr},
		{testUintVal, testUintValStr},
		{testUint8Val, testUint8ValStr},
		{testUint16Val, testUint16ValStr},
		{testUint32Val, testUint32ValStr},
		{testUint64Val, testUint64ValStr},
		{testFloat32Val, testFloat32ValStr},
		{testFloat64Val, testFloat64ValStr},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, toString(test.input))
	}

	assert.Panics(t, func() {
		toString(testInvalidVal)
	})
}

func TestToJSON(t *testing.T) {
	v := map[string]interface{}{
		"num": math.Inf(1),
	}
	_, err := jsonMarshal(v, "", "", true)
	if assert.Error(t, err) {
		assert.Equal(t, "{}", toJSON(v, "", "", true))
	}
}
