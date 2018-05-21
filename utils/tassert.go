package utils

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

func TAssert(matcher func(m []interface{}) error, test ...interface{}) {
	err := matcher(test)
	if err != nil {
		log.Fatalf("\x1b[91mERROR: %s\x1b[0m", err.Error())
	}

	log.Println("\x1b[32m\u2713\x1b[0m")
}

func IsNil(m []interface{}) error {
	if m[0] != nil {
		return fmt.Errorf("Expected nil, got %+v", m[0])
	}

	return nil
}

func Equals(m []interface{}) error {
	if !reflect.DeepEqual(m[0], m[1]) {
		return fmt.Errorf("Expected %+v, got %+v", m[1], m[0])
	}

	return nil
}

func ContainsSubstring(m []interface{}) error {
	actual, ok := m[0].(string)
	if !ok {
		return fmt.Errorf("Cannot convert %+v to string", m[0])
	}

	expected, ok := m[1].(string)
	if !ok {
		return fmt.Errorf("Cannot convert %+v to string", m[1])
	}

	if !strings.Contains(actual, expected) {
		return fmt.Errorf("Expected %s to contain %s", actual, expected)
	}

	return nil
}
