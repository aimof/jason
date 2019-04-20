// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Jason is designed to be convenient for reading arbitrary JSON while still honoring the strictness of the language.
// Inspired by other libraries and improved to work well for common use cases.
// It currently focuses on reading JSON data rather than creating it.
//
// Examples
//
// JSON is a commonly used data transfer format, so usually the data you want to read comes either as bytes or as an io.Reader.
//
// Create an object from bytes:
//		v, err := jason.NewObjectFromBytes(b)
// .. or from a net/http response body:
//		v, err := jason.NewObjectFromReader(res.body)
//
// Read values
//
// Reading values is done with Get<Type>(keys ...) or the generic Get(keys ...).
// If the key path is invalid or the type doesn't match, it will return an error and the default value.
//
//		name, err := v.GetString("name")
//		age, err := v.GetNumber("age")
//		verified, err := v.GetBoolean("verified")
//		education, err := v.GetObject("education")
//		friends, err := v.GetObjectArray("friends")
//
// Loop through array
//
// Getting an array is done by Get<Type>Array() or the generic GetValueArray(). It returns an error if the value at that keypath is null (or something else than the type).
//
//		friends, err := person.GetObjectArray("friends")
//		for _, friend := range friends {
//			name, err := friend.GetString("name")
//			age, err := friend.GetNumber("age")
//		}
//
// Loop through keys of object
//
// Looping through an object is done by first getting it with `GetObject()` and then range on the Map().
// The GetObject() method returns an error if the value at that keypath is null (or something else than an object).
//
//		person, err := person.GetObject("person")
//		for key, value := range person.Map() {
//		  ...
//		}
package jason

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// Error values returned when validation functions fail
var (
	ErrNotNull        = errors.New("is not null")
	ErrNotArray       = errors.New("Not an array")
	ErrNotNumber      = errors.New("not a number")
	ErrNotBool        = errors.New("no bool")
	ErrNotObject      = errors.New("not an object")
	ErrNotObjectArray = errors.New("not an object array")
	ErrNotString      = errors.New("not a string")
)

type KeyNotFoundError struct {
	Key string
}

func (k KeyNotFoundError) Error() string {
	if k.Key != "" {
		return fmt.Sprintf("key '%s' not found", k.Key)
	}

	return "key not found"
}

// Value represents an arbitrary JSON value.
// It may contain a bool, number, string, object, array or null.
type Value struct {
	data   interface{}
	exists bool  // Used to separate nil and non-existing values
	Err    error // True when the value is invalid.
}

// Object represents an object JSON object.
// It inherets from Value but with an additional method to access
// a map representation of it's content. It's useful when iterating.
type Object struct {
	Value
	m     map[string]*Value
	valid bool
}

type Array struct {
	Value
	s []*Value
}

// Marshal into bytes.
func (v *Object) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.m)
}

// Returns the golang map.
// Needed when iterating through the values of the object.
func (v *Object) Map() map[string]*Value {
	return v.m
}

func NewValue(reader io.Reader) (*Value, error) {
	v, err := newValueFromReader(reader)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func dataToObject(src *Value) (dist *Value) {
	switch src.data.(type) {
	case map[string]interface{}:
		dist := new(Object)
		for key, v := range src.data.(map[string]interface{}) {
			if v == nil {
				dist.m[key] = &Value{data: nil, exists: false}
			}
			switch v.(type) {
			case map[string]interface{}:
				dist.data.(map[string]interface{})[key] = &Value{data: dataToObject(&Value{data: v})}
			default:
				dist.data = &Value{data: v}
			}
		}
	case []interface{}:
		dist := make([]*Value, len(src.data.([]interface{})))
		for i, v := range src.data.([]interface{}) {
			dist[i].data = &Value{data: dataToObject(&Value{data: v})}
		}
	}
	return dist
}

func (parent *Value) Get(i interface{}) *Value {
	if parent == nil {
		return &Value{
			Err: fmt.Errorf("Get %v: parent is nil", i),
		}
	}
	if parent.Err != nil {
		return &*parent
	}
	switch i.(type) {
	case string:
		switch parent.Interface().(type) {
		case map[string]interface{}:
			child, ok := parent.Interface().(map[string]interface{})[i.(string)]
			if !ok {
				return &Value{Err: fmt.Errorf("Get: %v is not in keys", i)}
			}
			if child == nil {
				return &Value{data: nil, exists: false}
			}
			return &Value{data: child, exists: true}
		default:
			parent.Err = fmt.Errorf("Get %v: parent is not an object", i)
		}
	case int:
		switch parent.Interface().(type) {
		case []interface{}:
			if i.(int) < len(parent.Interface().([]interface{})) {
				child := parent.Interface().([]interface{})[i.(int)]
				if child == nil {
					return &Value{data: nil, exists: false}
				}
				return &Value{data: child, exists: true}
			}
			return &Value{Err: fmt.Errorf("Get %v: Index out of range", i)}
		}
	}
	return &Value{Err: fmt.Errorf("Get: %v is invalid", i)}
}

func (parent *Value) GetAll() (map[string]*Value, error) {
	switch parent.data.(type) {
	case map[string]interface{}:
	default:
		return nil, fmt.Errorf("GetAll: value is not an object, type: %v", reflect.TypeOf(parent.data))
	}

	children := make(map[string]*Value, len(parent.data.(map[string]interface{})))
	for key := range parent.data.(map[string]interface{}) {
		child := parent.Get(key)
		if child.err != nil {
			return nil, child.err
		}
		children[key] = child
	}
	return children, nil
}

// Creates a new value from an io.reader.
// Returns an error if the reader does not contain valid json.
// Useful for parsing the body of a net/http response.
// Example: NewFromReader(res.Body)
func NewValueFromReader(reader io.Reader) (*Value, error) {
	j, err := newValueFromReader(reader)
	switch j.Interface().(type) {
	case map[string]interface{}:
		j.data, err = objectFromValue(j, err)
		if err != nil {
			return nil, err
		}
	}
	return j, nil
}

func newValueFromReader(reader io.Reader) (*Value, error) {
	j := new(Value)
	d := json.NewDecoder(reader)
	d.UseNumber()
	err := d.Decode(&j.data)
	return j, err
}

// Duplicated
// Creates a new value from bytes.
// Returns an error if the bytes are not valid json.
func NewValueFromBytes(b []byte) (*Value, error) {
	r := bytes.NewReader(b)
	return NewValueFromReader(r)
}

func objectFromValue(v *Value, err error) (*Object, error) {
	if err != nil {
		return nil, err
	}

	o, err := v.Object()

	if err != nil {
		return nil, err
	}

	return o, nil
}

func NewObjectFromBytes(b []byte) (*Object, error) {
	return objectFromValue(newValueFromReader(bytes.NewReader(b)))
}

func NewObjectFromReader(reader io.Reader) (*Object, error) {
	return objectFromValue(newValueFromReader(reader))
}

// Marshal into bytes.
func (v *Value) Marshal() ([]byte, error) {
	return json.Marshal(v.data)
}

// Marshal into bytes
func (v *Value) MarshalJSON() ([]byte, error) {
	return v.Marshal()
}

// Get the interyling data as interface
func (v *Value) Interface() interface{} {
	return v.data
}

// Private Get
func (v *Value) get(key string) (*Value, error) {

	// Assume this is an object
	obj, err := v.Object()

	if err == nil {
		child, ok := obj.Map()[key]
		if ok {
			return child, nil
		} else {
			return nil, KeyNotFoundError{key}
		}
	}

	return nil, err
}

// Private get path
func (v *Value) getPath(keys []string) (*Value, error) {
	current := v
	var err error
	for _, key := range keys {
		current, err = current.get(key)

		if err != nil {
			return nil, err
		}
	}
	return current, nil
}

// Gets the value at key path.
// Returns error if the value does not exist.
// Consider using the more specific Get<Type>(..) methods instead.
// Example:
//		value, err := GetValue("address", "street")
func (v *Object) GetValue(keys ...string) (*Value, error) {
	return v.getPath(keys)
}

// Gets the value at key path and attempts to typecast the value into an object.
// Returns error if the value is not a json object.
// Example:
//		object, err := GetObject("person", "address")
func (v *Object) GetObject(keys ...string) (*Object, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return nil, err
	} else {

		obj, err := child.Object()

		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}

	}
}

// Gets the value at key path and attempts to typecast the value into a string.
// Returns error if the value is not a json string.
// Example:
//		string, err := GetString("address", "street")
func (v *Object) GetString(keys ...string) (string, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return "", err
	} else {
		return child.String()
	}
}

// Gets the value at key path and attempts to typecast the value into null.
// Returns error if the value is not json null.
// Example:
//		err := GetNull("address", "street")
func (v *Object) GetNull(keys ...string) error {
	child, err := v.getPath(keys)

	if err != nil {
		return err
	}

	return child.Null()
}

// Gets the value at key path and attempts to typecast the value into a number.
// Returns error if the value is not a json number.
// Example:
//		n, err := GetNumber("address", "street_number")
func (v *Object) GetNumber(keys ...string) (json.Number, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return "", err
	} else {

		n, err := child.Number()

		if err != nil {
			return "", err
		} else {
			return n, nil
		}
	}
}

// Gets the value at key path and attempts to typecast the value into a float64.
// Returns error if the value is not a json number.
// Example:
//		n, err := GetNumber("address", "street_number")
func (v *Object) GetFloat64(keys ...string) (float64, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return 0, err
	} else {

		n, err := child.Float64()

		if err != nil {
			return 0, err
		} else {
			return n, nil
		}
	}
}

// Gets the value at key path and attempts to typecast the value into a float64.
// Returns error if the value is not a json number.
// Example:
//		n, err := GetNumber("address", "street_number")
func (v *Object) GetInt64(keys ...string) (int64, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return 0, err
	} else {

		n, err := child.Int64()

		if err != nil {
			return 0, err
		} else {
			return n, nil
		}
	}
}

// Gets the value at key path and attempts to typecast the value into a float64.
// Returns error if the value is not a json number.
// Example:
//		v, err := GetInterface("address", "anything")
func (v *Object) GetInterface(keys ...string) (interface{}, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return nil, err
	} else {
		return child.Interface(), nil
	}
}

// Gets the value at key path and attempts to typecast the value into a bool.
// Returns error if the value is not a json boolean.
// Example:
//		married, err := GetBoolean("person", "married")
func (v *Object) GetBoolean(keys ...string) (bool, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return false, err
	}

	return child.Boolean()
}

// Gets the value at key path and attempts to typecast the value into an array.
// Returns error if the value is not a json array.
// Consider using the more specific Get<Type>Array() since it may reduce later type casts.
// Example:
//		friends, err := GetValueArray("person", "friends")
//		for i, friend := range friends {
//			... // friend will be of type Value here
//		}
func (v *Object) GetValueArray(keys ...string) ([]*Value, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return nil, err
	} else {

		return child.Array()

	}
}

// Gets the value at key path and attempts to typecast the value into an array of objects.
// Returns error if the value is not a json array or if any of the contained objects are not objects.
// Example:
//		friends, err := GetObjectArray("person", "friends")
//		for i, friend := range friends {
//			... // friend will be of type Object here
//		}
func (v *Object) GetObjectArray(keys ...string) ([]*Object, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return nil, err
	} else {

		array, err := child.Array()

		if err != nil {
			return nil, err
		} else {

			typedArray := make([]*Object, len(array))

			for index, arrayItem := range array {
				typedArrayItem, err := arrayItem.
					Object()

				if err != nil {
					return nil, err
				} else {
					typedArray[index] = typedArrayItem
				}

			}
			return typedArray, nil
		}
	}
}

// Gets the value at key path and attempts to typecast the value into an array of string.
// Returns error if the value is not a json array or if any of the contained objects are not strings.
// Gets the value at key path and attempts to typecast the value into an array of objects.
// Returns error if the value is not a json array or if any of the contained objects are not objects.
// Example:
//		friendNames, err := GetStringArray("person", "friend_names")
//		for i, friendName := range friendNames {
//			... // friendName will be of type string here
//		}
func (v *Object) GetStringArray(keys ...string) ([]string, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return nil, err
	} else {

		array, err := child.Array()

		if err != nil {
			return nil, err
		} else {

			typedArray := make([]string, len(array))

			for index, arrayItem := range array {
				typedArrayItem, err := arrayItem.String()

				if err != nil {
					return nil, err
				} else {
					typedArray[index] = typedArrayItem
				}

			}
			return typedArray, nil
		}
	}
}

// Gets the value at key path and attempts to typecast the value into an array of numbers.
// Returns error if the value is not a json array or if any of the contained objects are not numbers.
// Example:
//		friendAges, err := GetNumberArray("person", "friend_ages")
//		for i, friendAge := range friendAges {
//			... // friendAge will be of type float64 here
//		}
func (v *Object) GetNumberArray(keys ...string) ([]json.Number, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return nil, err
	} else {

		array, err := child.Array()

		if err != nil {
			return nil, err
		} else {

			typedArray := make([]json.Number, len(array))

			for index, arrayItem := range array {
				typedArrayItem, err := arrayItem.Number()

				if err != nil {
					return nil, err
				} else {
					typedArray[index] = typedArrayItem
				}

			}
			return typedArray, nil
		}
	}
}

// Gets the value at key path and attempts to typecast the value into an array of floats.
// Returns error if the value is not a json array or if any of the contained objects are not numbers.
func (v *Object) GetFloat64Array(keys ...string) ([]float64, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return nil, err
	} else {

		array, err := child.Array()

		if err != nil {
			return nil, err
		} else {

			typedArray := make([]float64, len(array))

			for index, arrayItem := range array {
				typedArrayItem, err := arrayItem.Float64()

				if err != nil {
					return nil, err
				} else {
					typedArray[index] = typedArrayItem
				}

			}
			return typedArray, nil
		}
	}
}

// Gets the value at key path and attempts to typecast the value into an array of ints.
// Returns error if the value is not a json array or if any of the contained objects are not numbers.
func (v *Object) GetInt64Array(keys ...string) ([]int64, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return nil, err
	} else {

		array, err := child.Array()

		if err != nil {
			return nil, err
		} else {

			typedArray := make([]int64, len(array))

			for index, arrayItem := range array {
				typedArrayItem, err := arrayItem.Int64()

				if err != nil {
					return nil, err
				} else {
					typedArray[index] = typedArrayItem
				}

			}
			return typedArray, nil
		}
	}
}

// Gets the value at key path and attempts to typecast the value into an array of bools.
// Returns error if the value is not a json array or if any of the contained objects are not booleans.
func (v *Object) GetBooleanArray(keys ...string) ([]bool, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return nil, err
	} else {

		array, err := child.Array()

		if err != nil {
			return nil, err
		} else {

			typedArray := make([]bool, len(array))

			for index, arrayItem := range array {
				typedArrayItem, err := arrayItem.Boolean()

				if err != nil {
					return nil, err
				} else {
					typedArray[index] = typedArrayItem
				}

			}
			return typedArray, nil
		}
	}
}

// Gets the value at key path and attempts to typecast the value into an array of nulls.
// Returns length, or an error if the value is not a json array or if any of the contained objects are not nulls.
func (v *Object) GetNullArray(keys ...string) (int64, error) {
	child, err := v.getPath(keys)

	if err != nil {
		return 0, err
	} else {

		array, err := child.Array()

		if err != nil {
			return 0, err
		} else {

			var length int64 = 0

			for _, arrayItem := range array {
				err := arrayItem.Null()

				if err != nil {
					return 0, err
				} else {
					length++
				}

			}
			return length, nil
		}
	}
}

// Returns an error if the value is not actually null
func (v *Value) Null() error {
	var valid bool

	// Check the type of this data
	switch v.data.(type) {
	case nil:
		valid = v.exists // Valid only if j also exists, since other values could possibly also be nil
		break
	}

	if valid {
		return nil
	}

	return ErrNotNull

}

// Attempts to typecast the current value into an array.
// Returns error if the current value is not a json array.
// Example:
//		friendsArray, err := friendsValue.Array()
func (v *Value) Array() ([]*Value, error) {
	var valid bool

	// Check the type of this data
	switch v.data.(type) {
	case []interface{}:
		valid = true
		break
	}

	// Unsure if this is a good way to use slices, it's probably not
	var slice []*Value

	if valid {

		for _, element := range v.data.([]interface{}) {
			child := Value{data: element, exists: true}
			slice = append(slice, &child)
		}

		return slice, nil
	}

	return slice, ErrNotArray

}

// Attempts to typecast the current value into a number.
// Returns error if the current value is not a json number.
// Example:
//		ageNumber, err := ageValue.Number()
func (v *Value) Number() (json.Number, error) {
	var valid bool

	// Check the type of this data
	switch v.data.(type) {
	case json.Number:
		valid = true
		break
	}

	if valid {
		return v.data.(json.Number), nil
	}

	return "", ErrNotNumber
}

// Attempts to typecast the current value into a float64.
// Returns error if the current value is not a json number.
// Example:
//		percentage, err := v.Float64()
func (v *Value) Float64() (float64, error) {
	n, err := v.Number()

	if err != nil {
		return 0, err
	}

	return n.Float64()
}

// Attempts to typecast the current value into a int64.
// Returns error if the current value is not a json number.
// Example:
//		id, err := v.Int64()
func (v *Value) Int64() (int64, error) {
	n, err := v.Number()

	if err != nil {
		return 0, err
	}

	return n.Int64()
}

// Attempts to typecast the current value into a bool.
// Returns error if the current value is not a json boolean.
// Example:
//		marriedBool, err := marriedValue.Boolean()
func (v *Value) Boolean() (bool, error) {
	var valid bool

	// Check the type of this data
	switch v.data.(type) {
	case bool:
		valid = true
		break
	}

	if valid {
		return v.data.(bool), nil
	}

	return false, ErrNotBool
}

// Attempts to typecast the current value into an object.
// Returns error if the current value is not a json object.
// Example:
//		friendObject, err := friendValue.Object()
func (v *Value) Object() (*Object, error) {

	var valid bool

	// Check the type of this data
	switch v.data.(type) {
	case map[string]interface{}:
		valid = true
		break
	}

	if valid {
		obj := new(Object)
		obj.valid = valid

		m := make(map[string]*Value)

		if valid {
			for key, element := range v.data.(map[string]interface{}) {
				m[key] = &Value{data: element, exists: true}

			}
		}

		obj.data = v.data
		obj.m = m

		return obj, nil
	}

	return nil, ErrNotObject
}

// Attempts to typecast the current value into an object arrau.
// Returns error if the current value is not an array of json objects
// Example:
//		friendObjects, err := friendValues.ObjectArray()
func (v *Value) ObjectArray() ([]*Object, error) {

	var valid bool

	// Check the type of this data
	switch v.data.(type) {
	case []interface{}:
		valid = true
		break
	}

	// Unsure if this is a good way to use slices, it's probably not
	var slice []*Object

	if valid {

		for _, element := range v.data.([]interface{}) {
			childValue := Value{data: element, exists: true}
			childObject, err := childValue.Object()

			if err != nil {
				return nil, ErrNotObjectArray
			}
			slice = append(slice, childObject)
		}

		return slice, nil
	}

	return nil, ErrNotObjectArray

}

// Attempts to typecast the current value into a string.
// Returns error if the current value is not a json string
// Example:
//		nameObject, err := nameValue.String()
func (v *Value) String() (string, error) {
	var valid bool

	// Check the type of this data
	switch v.data.(type) {
	case string:
		valid = true
		break
	}

	if valid {
		return v.data.(string), nil
	}

	return "", ErrNotString
}

// Returns the value a json formatted string.
// Note: The method named String() is used by golang's log method for logging.
// Example:
func (v *Object) String() string {

	f, err := json.Marshal(v.data)
	if err != nil {
		return err.Error()
	}

	return string(f)

}
