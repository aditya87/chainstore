package utils

import (
	"fmt"
	"log"
	"reflect"
)

func TAssert(test interface{}, matcher func(m ...interface{}) error, test2 ...interface{}) {
	err := matcher(test, test2)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("\u2713")
}

func IsNil(m ...interface{}) error {
	if m[0] != nil {
		return fmt.Errorf("Expected nil, got %+v", m)
	}

	return nil
}

func Equals(m ...interface{}) error {
	if !reflect.DeepEqual(m[0], m[1]) {
		return fmt.Errorf("Expected nil, got %+v", m)
	}

	return nil
}
