package hir

import "testing"

func TestTypeInfoEqualsBasic(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int"}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float"}
	voidType := TypeInfo{Kind: TypeKindVoid, Name: "void"}

	if !intType.Equals(TypeInfo{Kind: TypeKindInteger, Name: "int"}) {
		t.Error("int型等価性失敗")
	}

	if floatType.Equals(intType) {
		t.Error("float型とint型は等価ではない")
	}

	if !voidType.Equals(TypeInfo{Kind: TypeKindVoid, Name: "void"}) {
		t.Error("void型等価性失敗")
	}
}

func TestTypeInfoEqualsArray(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int"}
	arrayInt := TypeInfo{Kind: TypeKindArray, Name: "[]int", Parameters: []TypeInfo{intType}}
	arrayInt2 := TypeInfo{Kind: TypeKindArray, Name: "[]int", Parameters: []TypeInfo{{Kind: TypeKindInteger, Name: "int"}}}
	arrayFloat := TypeInfo{Kind: TypeKindArray, Name: "[]float", Parameters: []TypeInfo{{Kind: TypeKindFloat, Name: "float"}}}

	if !arrayInt.Equals(arrayInt2) {
		t.Error("int配列型等価性失敗")
	}

	if arrayInt.Equals(arrayFloat) {
		t.Error("int配列型とfloat配列型は等価ではない")
	}
}

func TestTypeInfoEqualsStruct(t *testing.T) {
	intField := FieldInfo{Name: "x", Type: TypeInfo{Kind: TypeKindInteger, Name: "int"}}
	floatField := FieldInfo{Name: "y", Type: TypeInfo{Kind: TypeKindFloat, Name: "float"}}

	structA := TypeInfo{
		Kind:   TypeKindStruct,
		Name:   "Point",
		Fields: []FieldInfo{intField, floatField},
	}
	structB := TypeInfo{
		Kind:   TypeKindStruct,
		Name:   "Point",
		Fields: []FieldInfo{intField, floatField},
	}
	structC := TypeInfo{
		Kind:   TypeKindStruct,
		Name:   "Position",
		Fields: []FieldInfo{intField, floatField},
	}

	if !structA.Equals(structB) {
		t.Error("同じ構造体型の等価性失敗")
	}

	if structA.Equals(structC) {
		t.Error("異なる名前の構造体型は等価ではない")
	}
}

func TestTypeInfoCanConvertToBasic(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int"}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float"}
	boolType := TypeInfo{Kind: TypeKindBoolean, Name: "bool"}
	voidType := TypeInfo{Kind: TypeKindVoid, Name: "void"}

	// 同じ型への変換は可能.
	if !intType.CanConvertTo(intType) {
		t.Error("同じ型への変換は可能であるべき")
	}

	// int <-> float の暗黙変換は可能.
	if !intType.CanConvertTo(floatType) {
		t.Error("int -> float 暗黙変換は可能であるべき")
	}

	if !floatType.CanConvertTo(intType) {
		t.Error("float -> int 暗黙変換は可能であるべき")
	}

	// int -> bool の変換は不可.
	if intType.CanConvertTo(boolType) {
		t.Error("int -> bool 変換は不可であるべき")
	}

	// void型への変換は不可.
	if intType.CanConvertTo(voidType) {
		t.Error("void型への変換は不可であるべき")
	}

	if voidType.CanConvertTo(intType) {
		t.Error("void型からの変換は不可であるべき")
	}
}

func TestTypeInfoCanConvertToArray(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int"}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float"}

	arrayInt := TypeInfo{Kind: TypeKindArray, Name: "[]int", Parameters: []TypeInfo{intType}}
	arrayFloat := TypeInfo{Kind: TypeKindArray, Name: "[]float", Parameters: []TypeInfo{floatType}}

	// 要素型が変換可能なら配列型も変換可能.
	if !arrayInt.CanConvertTo(arrayFloat) {
		t.Error("[]int -> []float 変換は可能であるべき")
	}

	if !arrayFloat.CanConvertTo(arrayInt) {
		t.Error("[]float -> []int 変換は可能であるべき")
	}
}

func TestTypeInfoCanConvertToFunction(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int"}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float"}

	funcIntToInt := TypeInfo{
		Kind:       TypeKindFunction,
		Name:       "func(int) int",
		Parameters: []TypeInfo{intType, intType}, // 引数と戻り値
	}
	funcFloatToFloat := TypeInfo{
		Kind:       TypeKindFunction,
		Name:       "func(float) float",
		Parameters: []TypeInfo{floatType, floatType}, // 引数と戻り値
	}

	// パラメータが変換可能なら関数型も変換可能.
	if !funcIntToInt.CanConvertTo(funcFloatToFloat) {
		t.Error("func(int) int -> func(float) float 変換は可能であるべき")
	}
}
