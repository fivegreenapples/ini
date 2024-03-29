package ini

import "reflect"

// dereference returns the underlying value and type that val points to if val
// is indeed a pointer. Otherwise dereference simply returns the original val
// and its assocaited type. A stack of pointers is dereferenced until the
// underlying non pointer value is found.
//
// isNilPointer will be true if val is a pointer type and it is nil.
func dereference(val reflect.Value) (rval reflect.Value, typ reflect.Type, isNilPointer bool) {

	rval = val
	typ = rval.Type()
	for rval.Kind() == reflect.Ptr {
		isNilPointer = rval.IsNil()
		typ = rval.Type().Elem()

		rval = rval.Elem()
	}

	return
}
