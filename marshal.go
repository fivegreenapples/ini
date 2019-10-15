package ini

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

// Marshal returns the INI encoding of v.
func Marshal(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, errors.New("can not marshal a nil value")
	}

	var err error
	canonical := newCanonical()
	origType := reflect.TypeOf(v)
	val, _, isNilPointer := dereference(reflect.ValueOf(v))

	if isNilPointer {
		return nil, errors.New("can not marshal a nil pointer")
	}

	switch val.Kind() {
	case reflect.Struct:
		err = marshalStruct(val, canonical, nil)

	case reflect.Map:
		err = marshalMap(val, canonical)

	default:
		err = fmt.Errorf("can not marshal unsupported type '%s'", origType)
	}

	if err != nil {
		return nil, err
	}

	return []byte(canonical.String()), nil
}

func marshalValue(name string, val reflect.Value, into *canonical, at *section) error {

	val, typ, isNilPointer := dereference(val)

	// switch on typ.Kind() not val.Kind() because val will be the zero value when
	// incoming val is a nil pointer.
	switch typ.Kind() {

	case reflect.Struct:
		if at != nil {
			return fmt.Errorf("can not marshal struct here - ini format does not support nested sections")
		}

		// Create a new section as necessary
		section := into.makeSection(name)

		// Marshal into it unless we have a nil pointer - in which case leave this
		// as an empty section.
		if !isNilPointer {
			structErr := marshalStruct(val, into, section)
			if structErr != nil {
				return fmt.Errorf("marshaling struct: %w", structErr)
			}
		}

	case reflect.Map:

		if at == nil {
			at = into.global
		}
		iter := val.MapRange()
		for iter.Next() {
			k, mapKeyErr := marshalScalarValue(iter.Key())
			if mapKeyErr != nil {
				return fmt.Errorf("marshaling map key: %w", mapKeyErr)
			}
			v, mapValErr := marshalScalarValue(iter.Value())
			if mapValErr != nil {
				return fmt.Errorf("marshaling map value: %w", mapValErr)
			}
			mapErr := at.addMapValue(name, k, v)
			if mapErr != nil {
				return fmt.Errorf("adding map value: %w", mapErr)
			}
		}

	case reflect.Slice, reflect.Array:

		if at == nil {
			at = into.global
		}
		for i := 0; i < val.Len(); i++ {
			scalarVal, scalarErr := marshalScalarValue(val.Index(i))
			if scalarErr != nil {
				return fmt.Errorf("marshaling slice element: %w", scalarErr)
			}
			arrayErr := at.addArrayValue(name, scalarVal)
			if arrayErr != nil {
				return fmt.Errorf("adding array value: %w", arrayErr)
			}
		}

	default:

		scalarVal, scalarErr := marshalScalarValue(val)
		if scalarErr != nil {
			return fmt.Errorf("marshaling scalar: %w", scalarErr)
		}
		if at == nil {
			at = into.global
		}
		scalarAddErr := at.addScalarValue(name, scalarVal)
		if scalarAddErr != nil {
			return fmt.Errorf("adding scalar value: %w", scalarAddErr)
		}
	}

	return nil
}

func marshalStruct(val reflect.Value, into *canonical, at *section) error {

	typ := val.Type()

	for f := 0; f < typ.NumField(); f++ {
		fld := typ.Field(f)
		isUnexported := fld.PkgPath != "" // Empty PkgPath implies unexported.
		if fld.Anonymous {
			//
			// Credit to Go team: The following block is lifted from json.Marshall code.
			//
			t := fld.Type
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			if isUnexported && t.Kind() != reflect.Struct {
				// Ignore embedded fields of unexported non-struct types.
				continue
			}
			// Do not ignore embedded fields of unexported struct types
			// since they may have exported fields.
			//
			// End lifting
			//
			structErr := marshalStruct(val.Field(f), into, at)
			if structErr != nil {
				return fmt.Errorf("marshaling embedded struct '%s': %w", fld.Name, structErr)
			}
		} else if isUnexported {
			// Ignore unexported non-embedded fields.
			continue
		} else {
			err := marshalValue(fld.Name, val.Field(f), into, at)
			if err != nil {
				return fmt.Errorf("marshaling field '%s': %w", fld.Name, err)
			}
		}
	}

	return nil
}

func marshalMap(v interface{}, into *canonical) error {
	return nil
}

func marshalScalarValue(val reflect.Value) (string, error) {

	origType := val.Type()
	val, _, isNilPointer := dereference(val)

	if isNilPointer {
		// Marshall nil pointer as empty string
		return "", nil
	}

	switch val.Kind() {
	case reflect.Bool:
		if val.Bool() {
			return "1", nil
		}
		return "0", nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(val.Uint(), 10), nil

	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'g', -1, 64), nil

	case reflect.String:
		return val.String(), nil
	}

	return "", fmt.Errorf("unsupported type '%s' as RHS value", origType)
}
