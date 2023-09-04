package binding

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
	"sync"
)

type StructValidator interface {
	// ValidateStruct 结构体验证，如果错误返回对应的错误信息
	ValidateStruct(any) error
	// Engine 返回对应使用的验证器
	Engine() any
}

var Validator StructValidator = &defaultValidator{}

type defaultValidator struct {
	one      sync.Once
	validate *validator.Validate
}

func (d *defaultValidator) ValidateStruct(obj any) error {
	if obj == nil {
		return nil
	}
	rv := reflect.ValueOf(obj)
	switch rv.Kind() {
	case reflect.Ptr:
		return d.ValidateStruct(rv.Elem().Interface())
	case reflect.Struct:
		return d.validateStruct(obj)
	case reflect.Slice, reflect.Array:
		l := rv.Len()
		validateRet := make(SliceValidationError, 0)
		for i := 0; i < l; i++ {
			if err := d.ValidateStruct(rv.Index(i).Interface()); err != nil {
				validateRet = append(validateRet, err)
			}
		}
		if len(validateRet) == 0 {
			return nil
		}
		return validateRet
	default:
		return nil
	}
}

func (d *defaultValidator) validateStruct(obj any) error {
	d.lazyInit()
	return d.validate.Struct(obj)
}

func (d *defaultValidator) lazyInit() {
	d.one.Do(func() {
		d.validate = validator.New()
	})
}

func (d *defaultValidator) Engine() interface{} {
	d.lazyInit()
	return d.validate
}

type SliceValidationError []error

func (err SliceValidationError) Error() string {
	n := len(err)
	switch n {
	case 0:
		return ""
	default:
		var b strings.Builder
		if err[0] != nil {
			fmt.Fprintf(&b, "[%d]: %s", 0, err[0].Error())
		}
		if n > 1 {
			for i := 1; i < n; i++ {
				if err[i] != nil {
					b.WriteString("\n")
					fmt.Fprintf(&b, "[%d]: %s", i, err[i].Error())
				}
			}
		}
		return b.String()
	}
}
