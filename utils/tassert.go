package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"strings"
	"time"
)

const debug = true

func dumpDebug() {
	if !debug {
		return
	}

	logs, err := ioutil.ReadFile("/app/agent.log")
	if err != nil {
		log.Printf("ERROR: Could not read logfile %v\n", err)
	}

	fmt.Println(string(logs))
}

func TAssert(matcher func(m []interface{}) error, test ...interface{}) {
	err := matcher(test)
	if err != nil {
		dumpDebug()
		log.Fatalf("\x1b[91mERROR: %s\x1b[0m", err.Error())
	}

	log.Println("\x1b[32m\u2713\x1b[0m")
}

func TAssertEventual(assertion func() bool, timeout ...int) {
	max := 10
	if len(timeout) != 0 {
		max = timeout[0]
	}
	for i := 0; i < max; i++ {
		test := assertion()
		if test {
			log.Println("\x1b[32m\u2022\u2713\x1b[0m")
			return
		}

		time.Sleep(1 * time.Second)
	}

	dumpDebug()
	log.Fatal("\x1b[91mERROR: eventual assertion failed\x1b[0m")
}

func IsNil(m []interface{}) error {
	if m[0] != nil {
		dumpDebug()
		return fmt.Errorf("Expected nil, got %+v", m[0])
	}

	return nil
}

func IsNotNil(m []interface{}) error {
	if m[0] == nil {
		dumpDebug()
		return fmt.Errorf("Expected non-nil, got %v", m[0])
	}

	return nil
}

func Equals(m []interface{}) error {
	if !reflect.DeepEqual(m[0], m[1]) {
		dumpDebug()
		return fmt.Errorf("Expected %+v, got %+v", m[1], m[0])
	}

	return nil
}

func ContainsSubstring(m []interface{}) error {
	actual, ok := m[0].(string)
	if !ok {
		dumpDebug()
		return fmt.Errorf("Cannot convert %+v to string", m[0])
	}

	expected, ok := m[1].(string)
	if !ok {
		dumpDebug()
		return fmt.Errorf("Cannot convert %+v to string", m[1])
	}

	if !strings.Contains(actual, expected) {
		dumpDebug()
		return fmt.Errorf("Expected %s to contain %s", actual, expected)
	}

	return nil
}
