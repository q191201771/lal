// Package assert 提供了单元测试时的断言功能
//
// 代码参考了 https://github.com/stretchr/testify
//
package assert

import (
	"bytes"
	"reflect"
)

type TestingT interface {
	Errorf(format string, args ...interface{})
}

func Equal(t TestingT, expected interface{}, actual interface{}, msg string) {
	if !equal(expected, actual) {
		t.Errorf("%s expected=%+v, actual=%+v", msg, expected, actual)
	}
	return
}

func isNil(actual interface{}) bool {
	if actual == nil {
		return true
	}
	v := reflect.ValueOf(actual)
	k := v.Kind()
	if k == reflect.Chan || k == reflect.Map || k == reflect.Ptr || k == reflect.Interface || k == reflect.Slice {
		return v.IsNil()
	}
	return false
}

func equal(expected, actual interface{}) bool {
	if expected == nil {
		return isNil(actual)
	}

	exp, ok := expected.([]byte)
	if !ok {
		return reflect.DeepEqual(expected, actual)
	}

	act, ok := actual.([]byte)
	if !ok {
		return false
	}
	//if exp == nil || act == nil {
	//	return exp == nil && act == nil
	//}
	return bytes.Equal(exp, act)
}