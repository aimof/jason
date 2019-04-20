package jason

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

var j = `[
	{
		"a": [
			{
				"3": 5
			},
			{
				"2": 3
			},
			{
				"b": [
					{
						"n": 2
					},
					{
						"b": "c"
					},
					{
						"c": [
							{
								"d": 1
							},
							{
								"u": 2
							}
						]
					}
				]
			}
		]
	},
	{
		"a": [
			{
				"3": 5
			},
			{
				"2": 3
			},
			{
				"b": [
					{
						"n": 2
					},
					{
						"b": "c"
					},
					{
						"c": [
							{
								"d": 1
							},
							{
								"f": [
									[
										2,
										3,
										4,
										{
											"tik": 1
										}
									],
									3,
									5
								]
							}
						]
					}
				]
			}
		]
	},
	{
		"a": [
			{
				"3": 5
			},
			{
				"2": 3
			},
			{
				"b": [
					{
						"n": 2
					},
					{
						"b": "c"
					},
					{
						"c": [
							{
								"d": 1
							},
							{
								"i": 2
							}
						]
					}
				]
			}
		]
	}
]`

func TestHighlyNestedJason(t *testing.T) {
	values, err := NewValueFromReader(strings.NewReader(j))
	if err != nil {
		t.Error(err)
		return
	}
	v := values.Get(1).Get("a").Get(2).Get("b").Get(2).Get("c").Get(1).Get("f").Get(0).Get(3).Get("tik")
	if v == nil {
		t.Error()
		return
	}
	if v.Err != nil {
		t.Error(v.Err)
		return
	}
	switch v.Interface().(type) {
	case json.Number:
		if num, err := v.Interface().(json.Number).Int64(); err != nil {
			t.Error()
		} else if num != 1 {
			t.Error()
		}
	default:
		t.Error(reflect.TypeOf(v.Interface()))
	}
}
