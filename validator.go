package structValidator

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"strconv"
	"strings"
)

var ErrNotStruct = errors.New("wrong argument given, should be a struct")
var ErrInvalidValidatorSyntax = errors.New("invalid validator syntax")
var ErrValidateForUnexportedFields = errors.New("validation for unexported field is not allowed")

type ValidationError struct {
	Err error
}

type ValidationErrors []ValidationError

// toStrings создает из ValidationErrors слайс из ошибок в строковом формате
func (v ValidationErrors) toStrings() []string {
	res := make([]string, len(v))
	for i, s := range v {
		res[i] = s.Err.Error()
	}
	return res
}

// Error возвращает все ошибки валидации в строковом формате.
// Каждая ошибка начинается с новой строки.
// Если ошибок нет - возвращается пустая строка.
func (v ValidationErrors) Error() string {
	return strings.Join(v.toStrings(), "\n")
}

// createReflectionSlice создает и возвращает слайс из reflect.Value.
// Если переданное значение не является слайсом или массивом, то
// в таком случае возвращается слайс, состоящий только из этого элемента
func createReflectionSlice(vv reflect.Value) []reflect.Value {
	v := make([]reflect.Value, 0)

	switch vv.Type().Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < vv.Len(); i++ {
			v = append(v, vv.Index(i))
		}
	default:
		v = append(v, vv)
	}
	return v
}

// checkLen проверяет, что длина всех переданных в него строк равна ln
func checkLen(vv reflect.Value, ln int) bool {
	v := createReflectionSlice(vv)

	for i := 0; i < len(v); i++ {
		if len(v[i].String()) != ln {
			return false
		}
	}
	return true
}

type comparator func(int64, int64) bool

// checkMinMax проверяет, что все элементы, содержащиеся в vv,
// удовлетворяют компаратору. Компаратор сравнивает значения с m и возвращает
// true, если все сравнения пройдены, и false в противном случае.
func checkMinMax(vv reflect.Value, m int, c comparator) bool {
	v := createReflectionSlice(vv)

	for i := 0; i < len(v); i++ {
		switch v[i].Kind() {
		case reflect.Int:
			if !c(int64(m), v[i].Int()) {
				return false
			}
		case reflect.String:
			if !c(int64(m), int64(len(v[i].String()))) {
				return false
			}
		}
	}
	return true
}

// checkContains проверяет, что все элементы, содержащиеся в vv,
// содержатся в слайсе c.
func checkContains(vv reflect.Value, c []reflect.Value) bool {
	v := createReflectionSlice(vv)

	for i := 0; i < len(v); i++ {
		ok := false
		for _, s := range c {
			switch v[i].Kind() {
			case reflect.Int:
				if v[i].Int() == s.Int() {
					ok = true
				}
			case reflect.String:
				if v[i].String() == s.String() {
					ok = true
				}
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// Validate производит валидацию публичных полей входной структуры на основе структурного тэга 'validate'
func Validate(v any) error {
	vt := reflect.TypeOf(v)
	vv := reflect.ValueOf(v)

	if vt.Kind() != reflect.Struct {
		return ErrNotStruct
	}

	errs := make(ValidationErrors, 0)

	for i, field := range reflect.VisibleFields(vt) {
		// если в структуре есть неэкспортируемое поле для валидации, то
		// возвращаем ошибку
		if s, ok := field.Tag.Lookup("validate"); !field.IsExported() && ok {
			errs = append(errs, ValidationError{ErrValidateForUnexportedFields})
			continue
		} else if ok {
			for _, b := range strings.Split(s, ";") {
				validator, toCheck := strings.Split(b, ":")[0], strings.Split(b, ":")[1]

				// в val будет лежать значение toCheck в числовом формате
				var val int

				switch validator {
				case "len", "min", "max":
					v, err := strconv.Atoi(toCheck)
					val = v
					if err != nil {
						errs = append(errs, ValidationError{ErrInvalidValidatorSyntax})
						continue
					}
				}

				switch validator {
				case "len":
					if !checkLen(vv.Field(i), val) {
						errs = append(errs, ValidationError{
							fmt.Errorf("field %s has an invalid length", field.Name)})
					}

				case "min":
					// компаратор для проверки на минимум
					c := func(min int64, v int64) bool {
						return v >= min
					}

					if !checkMinMax(vv.Field(i), val, c) {
						errs = append(errs, ValidationError{
							fmt.Errorf("field %s has value less than min", field.Name)})
					}

				case "max":
					// компаратор для проверки на максимум
					c := func(max int64, v int64) bool {
						return v <= max
					}

					if !checkMinMax(vv.Field(i), val, c) {
						errs = append(errs, ValidationError{
							fmt.Errorf("field %s has value bigger than max", field.Name)})
					}

				case "in":
					// в слайсе С будут лежать все допустимые значения, которые может содержать
					// поле vv.Field(i)
					c := make([]reflect.Value, 0)

					// rType будет содержать тип значения элемента/элементов в слайсе
					var rType reflect.Type
					switch vv.Field(i).Type().Kind() {
					case reflect.Array, reflect.Slice:
						rType = vv.Field(i).Index(0).Type()
					case reflect.Int, reflect.String:
						rType = vv.Field(i).Type()
					}

					for _, u := range strings.Split(toCheck, ",") {
						switch rType.Kind() {
						case reflect.Int:
							n, err := strconv.Atoi(u)
							if err != nil {
								errs = append(errs, ValidationError{ErrInvalidValidatorSyntax})
								continue
							}
							c = append(c, reflect.ValueOf(n))
						case reflect.String:
							c = append(c, reflect.ValueOf(u))
						}
					}

					if !checkContains(vv.Field(i), c) {
						errs = append(errs, ValidationError{
							fmt.Errorf("field %s does not occur in %v", field.Name,
								strings.Split(toCheck, ","))})
					}
				}
			}
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}
