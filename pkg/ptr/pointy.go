// Package ptr : a pointer package that can return a pointer of any object
//
// Example :
//				var s *string = ptr.Something("hello_mom")
//				var y *string = ptr[string].Something("hello_dad")
//				var o *string = ptr[SomeStruct].Something(SomeStruct{})
package ptr

// Get : returns a pointer for your variable declaration
func Get[T any](val T) *T {
	return &val
}

// Default : saves you from causing a nil pointer error by defaulting to a value if non existent
func Default[T any](val *T) *T {
	if val == nil {
		var d T
		return &d
	}
	return val
}

// Empty : returns an empty struct with pointer
func Empty[T any]() *T {
	var m T
	return &m
}

// EmptyNonPtr : returns an empty struct
func EmptyNonPtr[T any]() T {
	var m T
	return m
}
