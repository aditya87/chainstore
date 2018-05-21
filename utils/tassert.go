package utils

import (
	"fmt"
	"log"
	"reflect"
)

func TAssert(matcher func(m []interface{}) error, test ...interface{}) {
	err := matcher(test)
	if err != nil {
		log.Fatalf("\x1b[32m%s", err.Error())
	}

	log.Println("\x1b[32m\u2713")
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
