package hir

import "testing"

// TestStructLayout tests struct layout calculation.
func TestStructLayout(t *testing.T) {
	// Test simple struct with int and float fields.
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float", Size: 4, Alignment: 4}

	fields := []FieldInfo{
		{Name: "x", Type: intType},
		{Name: "y", Type: floatType},
	}

	layout := CalculateStructLayout(fields)

	if layout.Size != 8 {
		t.Errorf("Expected struct size 8, got %d", layout.Size)
	}

	if layout.Alignment != 4 {
		t.Errorf("Expected struct alignment 4, got %d", layout.Alignment)
	}

	if len(layout.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(layout.Fields))
	}

	// Check field offsets.
	if layout.Fields[0].Offset != 0 {
		t.Errorf("Expected field 0 offset 0, got %d", layout.Fields[0].Offset)
	}

	if layout.Fields[1].Offset != 4 {
		t.Errorf("Expected field 1 offset 4, got %d", layout.Fields[1].Offset)
	}
}

// TestStructLayoutWithPadding tests struct layout with padding requirements.
func TestStructLayoutWithPadding(t *testing.T) {
	// byte + int64 should have padding.
	byteType := TypeInfo{Kind: TypeKindInteger, Name: "byte", Size: 1, Alignment: 1}
	int64Type := TypeInfo{Kind: TypeKindInteger, Name: "int64", Size: 8, Alignment: 8}

	fields := []FieldInfo{
		{Name: "flag", Type: byteType},
		{Name: "value", Type: int64Type},
	}

	layout := CalculateStructLayout(fields)

	if layout.Size != 16 { // 1 + 7 padding + 8 = 16
		t.Errorf("Expected struct size 16, got %d", layout.Size)
	}

	if layout.Alignment != 8 {
		t.Errorf("Expected struct alignment 8, got %d", layout.Alignment)
	}

	// Check field offsets.
	if layout.Fields[0].Offset != 0 {
		t.Errorf("Expected field 0 offset 0, got %d", layout.Fields[0].Offset)
	}

	if layout.Fields[1].Offset != 8 { // Aligned to 8-byte boundary
		t.Errorf("Expected field 1 offset 8, got %d", layout.Fields[1].Offset)
	}
}

// TestCreateStructType tests struct type creation.
func TestCreateStructType(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float", Size: 4, Alignment: 4}

	fields := []FieldInfo{
		{Name: "x", Type: intType},
		{Name: "y", Type: floatType},
	}

	structType := CreateStructType("Point", fields)

	if structType.Kind != TypeKindStruct {
		t.Errorf("Expected struct kind, got %v", structType.Kind)
	}

	if structType.Name != "Point" {
		t.Errorf("Expected name 'Point', got %s", structType.Name)
	}

	if structType.Size != 8 {
		t.Errorf("Expected size 8, got %d", structType.Size)
	}

	if len(structType.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(structType.Fields))
	}
}

// TestEnumLayout tests algebraic data type layout calculation.
func TestEnumLayout(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	stringType := TypeInfo{Kind: TypeKindString, Name: "string", Size: 16, Alignment: 8}

	variants := []EnumVariant{
		{
			Name:   "None",
			Tag:    0,
			Fields: []FieldInfo{}, // No associated data
		},
		{
			Name:   "SomeInt",
			Tag:    1,
			Fields: []FieldInfo{{Name: "value", Type: intType}},
		},
		{
			Name:   "SomeString",
			Tag:    2,
			Fields: []FieldInfo{{Name: "value", Type: stringType}},
		},
	}

	layout := CalculateEnumLayout(variants, 8)

	// Size should be tag (8) + largest variant (16 for string) = 24.
	if layout.TotalSize != 24 {
		t.Errorf("Expected enum size 24, got %d", layout.TotalSize)
	}

	if layout.Alignment != 8 {
		t.Errorf("Expected enum alignment 8, got %d", layout.Alignment)
	}

	if layout.TagSize != 8 {
		t.Errorf("Expected tag size 8, got %d", layout.TagSize)
	}
}

// TestCreateEnumType tests enum type creation.
func TestCreateEnumType(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}

	variants := []EnumVariant{
		{Name: "None", Tag: 0, Fields: []FieldInfo{}},
		{Name: "Some", Tag: 1, Fields: []FieldInfo{{Name: "value", Type: intType}}},
	}

	enumType := CreateEnumType("Option", variants)

	if enumType.Name != "Option" {
		t.Errorf("Expected name 'Option', got %s", enumType.Name)
	}

	if len(enumType.Parameters) != 2 { // 2 variants
		t.Errorf("Expected 2 variants, got %d", len(enumType.Parameters))
	}

	if enumType.Size <= 0 {
		t.Errorf("Expected positive size, got %d", enumType.Size)
	}
}

// TestTupleLayout tests tuple layout calculation.
func TestTupleLayout(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float", Size: 4, Alignment: 4}
	int64Type := TypeInfo{Kind: TypeKindInteger, Name: "int64", Size: 8, Alignment: 8}

	elementTypes := []TypeInfo{intType, floatType, int64Type}

	layout := CalculateTupleLayout(elementTypes)

	// int(4) + float(4) + padding(0) + int64(8) = 16.
	if layout.TotalSize != 16 {
		t.Errorf("Expected tuple size 16, got %d", layout.TotalSize)
	}

	if layout.Alignment != 8 { // Largest alignment
		t.Errorf("Expected tuple alignment 8, got %d", layout.Alignment)
	}

	if len(layout.ElementLayouts) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(layout.ElementLayouts))
	}
}

// TestCreateTupleType tests tuple type creation.
func TestCreateTupleType(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	stringType := TypeInfo{Kind: TypeKindString, Name: "string", Size: 16, Alignment: 8}

	elementTypes := []TypeInfo{intType, stringType}

	tupleType := CreateTupleType("(int, string)", elementTypes)

	if tupleType.Name != "(int, string)" {
		t.Errorf("Expected name '(int, string)', got %s", tupleType.Name)
	}

	if len(tupleType.Parameters) != 2 {
		t.Errorf("Expected 2 elements, got %d", len(tupleType.Parameters))
	}

	if tupleType.Size <= 0 {
		t.Errorf("Expected positive size, got %d", tupleType.Size)
	}
}
