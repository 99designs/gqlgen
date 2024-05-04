package model

type IntTyped int

const (
	IntTypedOne IntTyped = iota + 1
	IntTypedTwo
)

const (
	IntUntypedOne = iota + 1
	IntUntypedTwo
)

type StringTyped string

const (
	StringTypedOne StringTyped = "ONE"
	StringTypedTwo StringTyped = "TWO"
)

const (
	StringUntypedOne = "ONE"
	StringUntypedTwo = "TWO"
)

type BoolTyped bool

const (
	BoolTypedTrue  BoolTyped = true
	BoolTypedFalse BoolTyped = false
)

const (
	BoolUntypedTrue  = true
	BoolUntypedFalse = false
)

type VarTyped bool

var (
	VarTypedTrue  VarTyped = true
	VarTypedFalse VarTyped = false
)

var (
	VarUntypedTrue  = true
	VarUntypedFalse = false
)
