package jason

import (
	"bytes"
	"encoding/json"
	"log"
	"reflect"
	"strings"
	"testing"
)

// ---
// I don't want to use Assert and True.
// So, new test doesn't use assert.
// ---

func TestNewValueRecursiveFromReader(t *testing.T) {
	j = `{
		"Foo": {
			"Bar": {
				"Fizz":[
					"Buzz"
				]
			}
		},
		"Bar": [
			{
				"Foo": "Bar"
			},
			3,
			"Fizz"
		]
	}`

	values, err := NewValue(strings.NewReader(j))
	if err != nil {
		t.Error(err)
		return
	}
	v := values.Get("Foo").Get("Bar").Get("Fizz").Get(0)
	if v == nil {
		t.Error()
		return
	}
	if v.err != nil {
		t.Error(v.err)
		return
	}
	switch v.Interface().(type) {
	case string:
		if v.Interface().(string) != "Buzz" {
			t.Error(v)
		}
	default:
		t.Error(reflect.TypeOf(v.Interface()))
	}
	v = values.Get("Bar").Get(1)
	switch v.Interface().(type) {
	case json.Number:
		if num, err := v.Interface().(json.Number).Int64(); err != nil {
			t.Error()
		} else if num != 3 {
			t.Error()
		}
	default:
		t.Error(reflect.TypeOf(v.Interface()))
	}
}

func TestGetAll(t *testing.T) {
	j = `{
		"Foo": null,
		"Bar": 4.5,
		"Fizz": "Buzz"
	}`

	v, err := NewValue(strings.NewReader(j))
	if err != nil {
		t.Error(err)
		return
	}

	got, err := v.GetAll()
	if err != nil {
		t.Error(err)
		return
	}
	if len(got) != 3 {
		t.Error()
	}
	for key, value := range got {
		switch key {
		case "Foo":
			if err := value.Null(); err != nil {
				t.Error()
			}
		case "Bar":
			if num, err := value.Float64(); err != nil {
				t.Error()
			} else if num != 4.5 {
				t.Error()
			}
		case "Fizz":
			if str, err := value.String(); err != nil {
				t.Error()
			} else if str != "Buzz" {
				t.Error()
			}
		default:
			t.Error()
		}
	}
}

func TestNewValueWithObject(t *testing.T) {
	jsonValues := make([]*Value, 0, 2)

	jsonBytesObjects := []byte(`{"Hello": "world!"}`)
	j, err := NewValueFromBytes(jsonBytesObjects)
	if err != nil {
		t.Error(err)
		return
	}
	jsonValues = append(jsonValues, j)

	j, err = NewValueFromReader(bytes.NewReader(jsonBytesObjects))
	if err != nil {
		t.Error(err)
		return
	}
	jsonValues = append(jsonValues, j)

	for _, v := range jsonValues {
		switch v.Interface().(type) {
		case *Object:
		default:
			t.Error("The value is not a object.")
			t.Log(reflect.TypeOf(v.Interface()))
		}
	}
}

// --- old test ---

type Assert struct {
	T *testing.T
}

func NewAssert(t *testing.T) *Assert {
	return &Assert{
		T: t,
	}
}

func (assert *Assert) True(value bool, message string) {
	if value == false {
		log.Panicln("Assert: ", message)
	}
}

func TestFirst(t *testing.T) {

	assert := NewAssert(t)

	testJSON := `{
    "name": "anton",
    "age": 29,
    "nothing": null,
    "true": true,
    "false": false,
    "list": [
      "first",
      "second"
    ],
    "list2": [
      {
        "street": "Street 42",
        "city": "Stockholm"
      },
      {
        "street": "Street 42",
        "city": "Stockholm"
      }
    ],
    "address": {
      "street": "Street 42",
      "city": "Stockholm"
    },
    "country": {
      "name": "Sweden"
    }
  }`

	j, err := NewObjectFromBytes([]byte(testJSON))
	if err != nil {
		t.Error(err)
		return
	}

	a, err := j.GetObject("address")
	assert.True(a != nil && err == nil, "failed to create json from string")

	assert.True(err == nil, "failed to create json from string")

	s, err := j.GetString("name")

	assert.True(s == "anton" && err == nil, "name should be a string")

	s, err = j.GetString("name")
	assert.True(s == "anton" && err == nil, "name shoud match")

	s, err = j.GetString("address", "street")
	assert.True(s == "Street 42" && err == nil, "street shoud match")
	//log.Println("s: ", s.String())

	_, err = j.GetNumber("age")
	assert.True(err == nil, "age should be a number")

	n, err := j.GetInt64("age")
	assert.True(n == 29 && err == nil, "age mismatch")

	ageInterface, err := j.GetInterface("age")
	assert.True(ageInterface != nil, "should be defined")
	assert.True(err == nil, "age interface error")

	invalidInterface, err := j.GetInterface("not_existing")
	assert.True(invalidInterface == nil, "should not give error here")
	assert.True(err != nil, "should give error here")

	age, err := j.GetValue("age")
	assert.True(age != nil && err == nil, "age should exist")

	age2, err := j.GetValue("age2")
	assert.True(age2 == nil && err != nil, "age2 should not exist")

	address, err := j.GetObject("address")
	assert.True(address != nil && err == nil, "address should be an object")

	//log.Println("address: ", address)

	s, err = address.GetString("street")

	addressAsString, err := j.GetString("address")
	assert.True(addressAsString == "" && err != nil, "address should not be an string")

	s, err = j.GetString("address", "street")
	assert.True(s == "Street 42" && err == nil, "street mismatching")

	s, err = j.GetString("address", "name2")
	assert.True(s == "" && err != nil, "nonexistent string fail")

	b, err := j.GetBoolean("true")
	assert.True(b == true && err == nil, "bool true test")

	b, err = j.GetBoolean("false")
	assert.True(b == false && err == nil, "bool false test")

	b, err = j.GetBoolean("invalid_field")
	assert.True(b == false && err != nil, "bool invalid test")

	list, err := j.GetValueArray("list")
	assert.True(list != nil && err == nil, "list should be an array")

	list2, err := j.GetValueArray("list2")
	assert.True(list2 != nil && err == nil, "list2 should be an array")

	list2Array, err := j.GetValueArray("list2")
	assert.True(err == nil, "List2 should not return error on AsArray")
	assert.True(len(list2Array) == 2, "List2 should should have length 2")

	list2Value, err := j.GetValue("list2")
	assert.True(err == nil, "List2 should not return error on value")

	list2ObjectArray, err := list2Value.ObjectArray()
	assert.True(err == nil, "list2Value should not return error on ObjectArray")
	assert.True(len(list2ObjectArray) == 2, "list2ObjectArray should should have length 2")

	for _, elementValue := range list2Array {
		//assert.True(element.IsObject() == true, "first fail")

		element, err := elementValue.Object()

		s, err = element.GetString("street")
		assert.True(s == "Street 42" && err == nil, "second fail")
	}

	obj, err := j.GetObject("country")
	assert.True(obj != nil && err == nil, "country should not return error on AsObject")
	for key, value := range obj.Map() {

		assert.True(key == "name", "country name key incorrect")

		s, err = value.String()
		assert.True(s == "Sweden" && err == nil, "country name should be Sweden")
	}
}

func TestSecond(t *testing.T) {
	json := `
  {
   "data": [
      {
         "id": "X999_Y999",
         "from": {
            "name": "Tom Brady", "id": "X12"
         },
         "message": "Looking forward to 2010!",
         "actions": [
            {
               "name": "Comment",
               "link": "http://www.facebook.com/X999/posts/Y999"
            },
            {
               "name": "Like",
               "link": "http://www.facebook.com/X999/posts/Y999"
            }
         ],
         "type": "status",
         "created_time": "2010-08-02T21:27:44+0000",
         "updated_time": "2010-08-02T21:27:44+0000"
      },
      {
         "id": "X998_Y998",
         "from": {
            "name": "Peyton Manning", "id": "X18"
         },
         "message": "Where's my contract?",
         "actions": [
            {
               "name": "Comment",
               "link": "http://www.facebook.com/X998/posts/Y998"
            },
            {
               "name": "Like",
               "link": "http://www.facebook.com/X998/posts/Y998"
            }
         ],
         "type": "status",
         "created_time": "2010-08-02T21:27:44+0000",
         "updated_time": "2010-08-02T21:27:44+0000"
      }
   ]
  }`

	assert := NewAssert(t)
	j, err := NewObjectFromBytes([]byte(json))

	if err != nil {
		t.Error(err)
		return
	}

	assert.True(j != nil && err == nil, "failed to parse json")

	dataObject, err := j.GetObject("data")
	assert.True(dataObject == nil && err != nil, "data should not be an object")

	dataArray, err := j.GetObjectArray("data")
	assert.True(dataArray != nil && err == nil, "data should be an object array")

	for index, dataItem := range dataArray {

		if index == 0 {
			id, err := dataItem.GetString("id")
			assert.True(id == "X999_Y999" && err == nil, "item id mismatch")

			fromName, err := dataItem.GetString("from", "name")
			assert.True(fromName == "Tom Brady" && err == nil, "fromName mismatch")

			actions, err := dataItem.GetObjectArray("actions")

			for index, action := range actions {

				if index == 1 {
					name, err := action.GetString("name")
					assert.True(name == "Like" && err == nil, "name mismatch")

					link, err := action.GetString("link")
					assert.True(link == "http://www.facebook.com/X999/posts/Y999" && err == nil, "Like mismatch")

				}

			}
		} else if index == 1 {
			id, err := dataItem.GetString("id")
			assert.True(id == "X998_Y998" && err == nil, "item id mismatch")
		}

	}

}

func TestErrors(t *testing.T) {
	json := `
  {
    "string": "hello",
    "number": 1,
    "array": [1,2,3]
  }`

	errstr := "expected an error getting %s, but got '%s'"

	j, err := NewObjectFromBytes([]byte(json))
	if err != nil {
		t.Fatal("failed to parse json")
	}

	if _, err = j.GetObject("string"); err != ErrNotObject {
		t.Errorf(errstr, "object", err)
	}

	if err = j.GetNull("string"); err != ErrNotNull {
		t.Errorf(errstr, "null", err)
	}

	if _, err = j.GetStringArray("string"); err != ErrNotArray {
		t.Errorf(errstr, "array", err)
	}

	if _, err = j.GetStringArray("array"); err != ErrNotString {
		t.Errorf(errstr, "string array", err)
	}

	if _, err = j.GetNumber("array"); err != ErrNotNumber {
		t.Errorf(errstr, "number", err)
	}

	if _, err = j.GetBoolean("array"); err != ErrNotBool {
		t.Errorf(errstr, "boolean", err)
	}

	if _, err = j.GetString("number"); err != ErrNotString {
		t.Errorf(errstr, "string", err)
	}

	_, err = j.GetString("not_found")
	if e, ok := err.(KeyNotFoundError); !ok {
		t.Errorf(errstr, "key not found error", e)
	}

}
