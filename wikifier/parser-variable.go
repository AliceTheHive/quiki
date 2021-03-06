package wikifier

import "regexp"

type variableName struct {
	parent catch
	*genericCatch
}

func newVariableName(pfx string, pos Position) *variableName {
	pc := []posContent{{pfx, pos}}
	return &variableName{genericCatch: &genericCatch{positionedPrefix: pc}}
}

func (vn *variableName) catchType() catchType {
	return catchTypeVariableName
}

func (vn *variableName) parentCatch() catch {
	return vn.parent
}

// word-like chars and periods are OK in var names
func (vn *variableName) byteOK(b byte) bool {
	ok, _ := regexp.Match(`[\w\.\/]`, []byte{b})
	return ok
}

// skip whitespace in variable name
func (vn *variableName) shouldSkipByte(b byte) bool {
	skip, _ := regexp.Match(`\s`, []byte{b})
	return skip
}

type variableValue struct {
	parent catch
	*genericCatch
}

func newVariableValue() *variableValue {
	return &variableValue{genericCatch: &genericCatch{}}
}

func (vv *variableValue) catchType() catchType {
	return catchTypeVariableValue
}

func (vv *variableValue) parentCatch() catch {
	return vv.parent
}

func (vv *variableValue) byteOK(b byte) bool {
	if b == '\n' {
		return true
	}
	ok, _ := regexp.Match(`.`, []byte{b})
	return ok
}

// skip whitespace in variable name
func (vv *variableValue) shouldSkipByte(b byte) bool {
	return false
}
