package main

import (
	"fmt"
	"gopkg.in/go-playground/validator.v8"
)

type Test struct {
	MyInt *int `validate:"omitempty,gte=2,lte=255"`
}
type Test2 struct {
	MyInt *int `validate:"gte=2,lte=255"`
}

func main() {
	val1 := 0
	val2 := 256
	t1 := Test{MyInt: &val1}
	t2 := Test{MyInt: &val2}
	t3 := Test{MyInt: nil}

	config := &validator.Config{TagName: "validate"}
	validate := validator.New(config)
	err := validate.Struct(t1)
	fmt.Printf("t1: %v\n", err)
	err = validate.Struct(t2)
	fmt.Printf("t2: %v\n", err)
	err = validate.Struct(t3)
	fmt.Printf("t3: %v\n", err)
}