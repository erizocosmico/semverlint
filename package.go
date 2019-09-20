package semverlint

import "go/types"

// Package with all its exposed members.
type Package struct {
	Name       string
	Path       string
	Vars       []Var
	Consts     []Const
	Funcs      []Func
	Structs    []Struct
	Interfaces []Interface
	Types      []TypeDef
}

// TypeDef is a type definition of the type `type A B` or `type A = B`.
type TypeDef struct {
	Name  string
	Type  types.Type
	Alias bool
}

// Var is an exposed variable.
type Var struct {
	Name string
	Type types.Type
}

// Const is an exposed constant.
type Const struct {
	Name  string
	Type  types.Type
	Value string
}

// Func or method exposed.
type Func struct {
	Name   string
	Args   []types.Type
	Return []types.Type
}

// Interface exposed.
type Interface struct {
	Name    string
	Methods []Func
}

// Struct exposed.
type Struct struct {
	Name    string
	Fields  []Field
	Methods []Func
}

// Field exposed in a struct.
type Field struct {
	Name string
	Type types.Type
}
