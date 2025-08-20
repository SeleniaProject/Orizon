// Package types provides implementation of Orizon's core data types.
// These types form the foundation of the Orizon runtime and are essential.
// for self-hosting the compiler.
package types

import (
	"sync"
	"unsafe"

	"github.com/orizon-lang/orizon/internal/allocator"
)

// CoreTypeKind represents the kind of core type.
type CoreTypeKind int

const (
	CoreTypeOption CoreTypeKind = iota
	CoreTypeResult
	CoreTypeSlice
	CoreTypeString
	CoreTypeVec
)

// CoreTypeManager manages core type instances and operations.
type CoreTypeManager struct {
	allocator    allocator.Allocator
	stringPool   map[string]*OrizonString
	stringPoolMu sync.RWMutex // Protects stringPool from concurrent access
}

// GlobalCoreTypeManager is the global instance for core type management.
var GlobalCoreTypeManager *CoreTypeManager

// InitializeCoreTypes initializes the core type system.
func InitializeCoreTypes(alloc allocator.Allocator) error {
	GlobalCoreTypeManager = &CoreTypeManager{
		allocator:  alloc,
		stringPool: make(map[string]*OrizonString),
	}

	return nil
}

// ShutdownCoreTypes cleans up the core type system.
func ShutdownCoreTypes() {
	if GlobalCoreTypeManager != nil {
		// Clear string pool with proper synchronization.
		GlobalCoreTypeManager.stringPoolMu.Lock()
		for _, str := range GlobalCoreTypeManager.stringPool {
			str.Destroy()
		}

		GlobalCoreTypeManager.stringPool = nil
		GlobalCoreTypeManager.stringPoolMu.Unlock()
		GlobalCoreTypeManager = nil
	}
}

// Option<T> - A type that represents optional values.
type OrizonOption struct {
	value    unsafe.Pointer
	typeInfo *TypeInfo
	hasValue bool
}

// Option constructors.
func NewSome(value unsafe.Pointer, typeInfo *TypeInfo) *OrizonOption {
	return &OrizonOption{
		hasValue: true,
		value:    value,
		typeInfo: typeInfo,
	}
}

func NewNone(typeInfo *TypeInfo) *OrizonOption {
	return &OrizonOption{
		hasValue: false,
		value:    nil,
		typeInfo: typeInfo,
	}
}

// Option methods.
func (opt *OrizonOption) IsSome() bool {
	return opt.hasValue
}

func (opt *OrizonOption) IsNone() bool {
	return !opt.hasValue
}

func (opt *OrizonOption) Unwrap() unsafe.Pointer {
	if !opt.hasValue {
		panic("Called unwrap on None value")
	}

	return opt.value
}

func (opt *OrizonOption) UnwrapOr(defaultValue unsafe.Pointer) unsafe.Pointer {
	if opt.hasValue {
		return opt.value
	}

	return defaultValue
}

func (opt *OrizonOption) Map(fn func(unsafe.Pointer) unsafe.Pointer) *OrizonOption {
	if opt.hasValue {
		newValue := fn(opt.value)

		return NewSome(newValue, opt.typeInfo)
	}

	return NewNone(opt.typeInfo)
}

// Result<T, E> - A type that represents either success (Ok) or failure (Err).
type OrizonResult struct {
	value   unsafe.Pointer
	error   unsafe.Pointer
	okType  *TypeInfo
	errType *TypeInfo
	isOk    bool
}

// Result constructors.
func NewOk(value unsafe.Pointer, okType *TypeInfo, errType *TypeInfo) *OrizonResult {
	return &OrizonResult{
		isOk:    true,
		value:   value,
		error:   nil,
		okType:  okType,
		errType: errType,
	}
}

func NewErr(error unsafe.Pointer, okType *TypeInfo, errType *TypeInfo) *OrizonResult {
	return &OrizonResult{
		isOk:    false,
		value:   nil,
		error:   error,
		okType:  okType,
		errType: errType,
	}
}

// Result methods.
func (res *OrizonResult) IsOk() bool {
	return res.isOk
}

func (res *OrizonResult) IsErr() bool {
	return !res.isOk
}

func (res *OrizonResult) Unwrap() unsafe.Pointer {
	if !res.isOk {
		panic("Called unwrap on Err result")
	}

	return res.value
}

func (res *OrizonResult) UnwrapErr() unsafe.Pointer {
	if res.isOk {
		panic("Called unwrap_err on Ok result")
	}

	return res.error
}

func (res *OrizonResult) UnwrapOr(defaultValue unsafe.Pointer) unsafe.Pointer {
	if res.isOk {
		return res.value
	}

	return defaultValue
}

func (res *OrizonResult) Map(fn func(unsafe.Pointer) unsafe.Pointer) *OrizonResult {
	if res.isOk {
		newValue := fn(res.value)

		return NewOk(newValue, res.okType, res.errType)
	}

	return res
}

func (res *OrizonResult) MapErr(fn func(unsafe.Pointer) unsafe.Pointer) *OrizonResult {
	if !res.isOk {
		newError := fn(res.error)

		return NewErr(newError, res.okType, res.errType)
	}

	return res
}

// Slice<T> - A slice type similar to Go slices.
type OrizonSlice struct {
	data     unsafe.Pointer
	typeInfo *TypeInfo
	length   uintptr
	capacity uintptr
}

// Slice constructors.
func NewSlice(data unsafe.Pointer, length, capacity uintptr, typeInfo *TypeInfo) *OrizonSlice {
	return &OrizonSlice{
		data:     data,
		length:   length,
		capacity: capacity,
		typeInfo: typeInfo,
	}
}

func NewSliceFromArray(array unsafe.Pointer, length uintptr, typeInfo *TypeInfo) *OrizonSlice {
	return &OrizonSlice{
		data:     array,
		length:   length,
		capacity: length,
		typeInfo: typeInfo,
	}
}

// Slice methods.
func (slice *OrizonSlice) Len() uintptr {
	return slice.length
}

func (slice *OrizonSlice) Cap() uintptr {
	return slice.capacity
}

func (slice *OrizonSlice) IsEmpty() bool {
	return slice.length == 0
}

func (slice *OrizonSlice) Get(index uintptr) unsafe.Pointer {
	if index >= slice.length {
		panic("Index out of bounds")
	}

	if slice.typeInfo == nil {
		panic("Invalid slice: nil typeInfo")
	}

	elementSize := slice.typeInfo.Size
	if elementSize == 0 {
		panic("Invalid element size: 0")
	}
	// Validate data pointer before pointer arithmetic.
	if slice.data == nil {
		panic("Null slice data pointer")
	}
	// Check for integer overflow in offset calculation (multiplication).
	if elementSize > 0 && index >= ^uintptr(0)/elementSize {
		panic("Offset calculation would overflow")
	}
	offset := index * elementSize
	// Check for integer overflow in pointer addition: base + offset exceeds uintptr max.
	base := uintptr(slice.data)
	if offset > ^uintptr(0)-base {
		panic("Pointer addition would overflow")
	}
	// Use unsafe.Add for race detector compatibility
	return unsafe.Add(slice.data, offset)
}

func (slice *OrizonSlice) Set(index uintptr, value unsafe.Pointer) {
	if index >= slice.length {
		panic("Index out of bounds")
	}

	if slice.typeInfo == nil {
		panic("Invalid slice: nil typeInfo")
	}

	elementSize := slice.typeInfo.Size
	if elementSize == 0 {
		panic("Invalid element size: 0")
	}
	// Validate input pointers for safety.
	if slice.data == nil {
		panic("Null slice data pointer")
	}

	if value == nil {
		panic("Null value pointer")
	}
	// Check for integer overflow in offset calculation (multiplication).
	if elementSize > 0 && index >= ^uintptr(0)/elementSize {
		panic("Offset calculation would overflow")
	}
	offset := index * elementSize
	// Check for integer overflow in pointer addition: base + offset exceeds uintptr max.
	base := uintptr(slice.data)
	if offset > ^uintptr(0)-base {
		panic("Pointer addition would overflow")
	}
	// Use unsafe.Add for race detector compatibility
	dest := unsafe.Add(slice.data, offset)
	// Copy value bytes to destination.
	copyBytes(dest, value, elementSize)
}

func (slice *OrizonSlice) Sub(start, end uintptr) *OrizonSlice {
	if start > end || end > slice.length {
		panic("Invalid slice bounds")
	}

	elementSize := slice.typeInfo.Size
	newData := unsafe.Add(slice.data, start*elementSize)
	newLength := end - start
	newCapacity := slice.capacity - start

	return NewSlice(newData, newLength, newCapacity, slice.typeInfo)
}

// String - UTF-8 string type.
type OrizonString struct {
	data   unsafe.Pointer
	length uintptr
	hash   uint64
}

// String constructors.
func NewString(data []byte) *OrizonString {
	if GlobalCoreTypeManager == nil {
		panic("Core type manager not initialized")
	}

	// Check string pool first with read lock.
	strKey := string(data)

	GlobalCoreTypeManager.stringPoolMu.RLock()
	if pooled, exists := GlobalCoreTypeManager.stringPool[strKey]; exists {
		GlobalCoreTypeManager.stringPoolMu.RUnlock()

		return pooled
	}
	GlobalCoreTypeManager.stringPoolMu.RUnlock()

	// Allocate memory for string data.
	length := uintptr(len(data))
	if length == 0 {
		str := &OrizonString{
			data:   nil,
			length: 0,
			hash:   0,
		}
		// Add to pool with write lock.
		GlobalCoreTypeManager.stringPoolMu.Lock()
		// Double-check pattern to avoid race condition.
		if pooled, exists := GlobalCoreTypeManager.stringPool[strKey]; exists {
			GlobalCoreTypeManager.stringPoolMu.Unlock()

			return pooled
		}

		GlobalCoreTypeManager.stringPool[strKey] = str
		GlobalCoreTypeManager.stringPoolMu.Unlock()

		return str
	}

	stringData := GlobalCoreTypeManager.allocator.Alloc(length)
	if stringData == nil {
		panic("Failed to allocate memory for string")
	}

	// Copy data.
	copyBytes(stringData, unsafe.Pointer(&data[0]), length)

	str := &OrizonString{
		data:   stringData,
		length: length,
		hash:   hashBytes(data),
	}

	// Add to pool with write lock.
	GlobalCoreTypeManager.stringPoolMu.Lock()
	// Double-check pattern to avoid race condition.
	if pooled, exists := GlobalCoreTypeManager.stringPool[strKey]; exists {
		// Free the allocated memory since we already have this string.
		GlobalCoreTypeManager.allocator.Free(stringData)
		GlobalCoreTypeManager.stringPoolMu.Unlock()

		return pooled
	}

	GlobalCoreTypeManager.stringPool[strKey] = str
	GlobalCoreTypeManager.stringPoolMu.Unlock()

	return str
}

func NewStringFromCString(cstr unsafe.Pointer) *OrizonString {
	length := cStringLength(cstr)
	data := make([]byte, length)
	copyBytes(unsafe.Pointer(&data[0]), cstr, length)

	return NewString(data)
}

// String methods.
func (str *OrizonString) Len() uintptr {
	return str.length
}

func (str *OrizonString) IsEmpty() bool {
	return str.length == 0
}

func (str *OrizonString) AsBytes() []byte {
	if str.length == 0 {
		return nil
	}

	bytes := make([]byte, str.length)
	copyBytes(unsafe.Pointer(&bytes[0]), str.data, str.length)

	return bytes
}

func (str *OrizonString) AsGoString() string {
	return string(str.AsBytes())
}

func (str *OrizonString) Hash() uint64 {
	return str.hash
}

func (str *OrizonString) Equals(other *OrizonString) bool {
	if str == other {
		return true
	}

	if str.length != other.length {
		return false
	}

	if str.hash != other.hash {
		return false
	}

	return compareBytes(str.data, other.data, str.length) == 0
}

func (str *OrizonString) Compare(other *OrizonString) int {
	minLen := str.length
	if other.length < minLen {
		minLen = other.length
	}

	if minLen > 0 {
		cmp := compareBytes(str.data, other.data, minLen)
		if cmp != 0 {
			return cmp
		}
	}

	if str.length < other.length {
		return -1
	} else if str.length > other.length {
		return 1
	}

	return 0
}

func (str *OrizonString) Concat(other *OrizonString) *OrizonString {
	if str.length == 0 {
		return other
	}

	if other.length == 0 {
		return str
	}

	newLength := str.length + other.length
	data := make([]byte, newLength)

	copyBytes(unsafe.Pointer(&data[0]), str.data, str.length)
	copyBytes(unsafe.Add(unsafe.Pointer(&data[0]), str.length), other.data, other.length)

	return NewString(data)
}

func (str *OrizonString) Destroy() {
	if str.data != nil && GlobalCoreTypeManager != nil {
		GlobalCoreTypeManager.allocator.Free(str.data)
		str.data = nil
		str.length = 0
	}
}

// Vec<T> - Dynamic array type.
type OrizonVec struct {
	data     unsafe.Pointer
	typeInfo *TypeInfo
	length   uintptr
	capacity uintptr
}

// Vec constructors.
func NewVec(typeInfo *TypeInfo) *OrizonVec {
	return &OrizonVec{
		data:     nil,
		length:   0,
		capacity: 0,
		typeInfo: typeInfo,
	}
}

func NewVecWithCapacity(capacity uintptr, typeInfo *TypeInfo) *OrizonVec {
	vec := &OrizonVec{
		data:     nil,
		length:   0,
		capacity: 0, // Start with 0, reserve will set it properly
		typeInfo: typeInfo,
	}

	if capacity > 0 {
		vec.reserve(capacity)
	}

	return vec
}

// Vec methods.
func (vec *OrizonVec) Len() uintptr {
	return vec.length
}

func (vec *OrizonVec) Cap() uintptr {
	return vec.capacity
}

func (vec *OrizonVec) IsEmpty() bool {
	return vec.length == 0
}

func (vec *OrizonVec) Get(index uintptr) unsafe.Pointer {
	if index >= vec.length {
		panic("Index out of bounds")
	}

	if vec.typeInfo == nil {
		panic("Invalid vector: nil typeInfo")
	}

	elementSize := vec.typeInfo.Size
	if elementSize == 0 {
		panic("Invalid element size: 0")
	}
	// Validate data pointer before pointer arithmetic.
	if vec.data == nil {
		panic("Null vector data pointer")
	}
	// Check for integer overflow in offset calculation.
	if elementSize > 0 && index > ^uintptr(0)/elementSize {
		panic("Offset calculation would overflow")
	}
	// Use unsafe.Add for race detector compatibility
	return unsafe.Add(vec.data, index*elementSize)
}

func (vec *OrizonVec) Set(index uintptr, value unsafe.Pointer) {
	if index >= vec.length {
		panic("Index out of bounds")
	}

	if vec.typeInfo == nil {
		panic("Invalid vector: nil typeInfo")
	}

	elementSize := vec.typeInfo.Size
	if elementSize == 0 {
		panic("Invalid element size: 0")
	}
	// Validate input pointers for safety.
	if vec.data == nil {
		panic("Null vector data pointer")
	}

	if value == nil {
		panic("Null value pointer")
	}
	// Check for integer overflow in offset calculation.
	if elementSize > 0 && index > ^uintptr(0)/elementSize {
		panic("Offset calculation would overflow")
	}
	// Use unsafe.Add for race detector compatibility
	dest := unsafe.Add(vec.data, index*elementSize)
	copyBytes(dest, value, elementSize)
}

func (vec *OrizonVec) Push(value unsafe.Pointer) {
	if value == nil {
		panic("Cannot push nil value")
	}

	// Ensure vec has capacity.
	if vec.capacity == 0 || vec.length >= vec.capacity {
		newCapacity := vec.capacity * 2
		if newCapacity == 0 {
			newCapacity = 4
		}

		vec.reserve(newCapacity)
	}

	if vec.data == nil {
		panic("Vector data is nil after reserve")
	}

	elementSize := vec.typeInfo.Size
	dest := unsafe.Add(vec.data, vec.length*elementSize)
	copyBytes(dest, value, elementSize)

	vec.length++
}

func (vec *OrizonVec) Pop() unsafe.Pointer {
	if vec.length == 0 {
		panic("Cannot pop from empty vector")
	}
	// Validate data pointer before operation.
	if vec.data == nil {
		panic("Null vector data pointer during pop")
	}

	vec.length--
	elementSize := vec.typeInfo.Size
	lastElement := unsafe.Add(vec.data, vec.length*elementSize)

	return lastElement
}

func (vec *OrizonVec) Clear() {
	vec.length = 0
}

func (vec *OrizonVec) AsSlice() *OrizonSlice {
	return NewSlice(vec.data, vec.length, vec.capacity, vec.typeInfo)
}

func (vec *OrizonVec) reserve(newCapacity uintptr) {
	if newCapacity <= vec.capacity {
		return
	}

	if GlobalCoreTypeManager == nil {
		panic("Core type manager not initialized")
	}

	if GlobalCoreTypeManager.allocator == nil {
		panic("Allocator not initialized")
	}

	elementSize := vec.typeInfo.Size
	newSize := newCapacity * elementSize

	if vec.data == nil {
		vec.data = GlobalCoreTypeManager.allocator.Alloc(newSize)
		if vec.data == nil {
			panic("Failed to allocate memory for vector - initial allocation failed")
		}
	} else {
		newData := GlobalCoreTypeManager.allocator.Realloc(vec.data, newSize)
		if newData == nil {
			panic("Failed to reallocate memory for vector")
		}

		vec.data = newData
	}

	vec.capacity = newCapacity
}

func (vec *OrizonVec) Destroy() {
	if vec.data != nil && GlobalCoreTypeManager != nil {
		GlobalCoreTypeManager.allocator.Free(vec.data)
		vec.data = nil
		vec.length = 0
		vec.capacity = 0
	}
}

// TypeInfo represents type information for core types.
type TypeInfo struct {
	Name        string
	Size        uintptr
	Alignment   uintptr
	Kind        CoreTypeKind
	IsPointer   bool
	IsPrimitive bool
}

// Common type info instances.
var (
	TypeInfoInt8    = &TypeInfo{Size: 1, Alignment: 1, Name: "i8", IsPrimitive: true}
	TypeInfoInt16   = &TypeInfo{Size: 2, Alignment: 2, Name: "i16", IsPrimitive: true}
	TypeInfoInt32   = &TypeInfo{Size: 4, Alignment: 4, Name: "i32", IsPrimitive: true}
	TypeInfoInt64   = &TypeInfo{Size: 8, Alignment: 8, Name: "i64", IsPrimitive: true}
	TypeInfoUInt8   = &TypeInfo{Size: 1, Alignment: 1, Name: "u8", IsPrimitive: true}
	TypeInfoUInt16  = &TypeInfo{Size: 2, Alignment: 2, Name: "u16", IsPrimitive: true}
	TypeInfoUInt32  = &TypeInfo{Size: 4, Alignment: 4, Name: "u32", IsPrimitive: true}
	TypeInfoUInt64  = &TypeInfo{Size: 8, Alignment: 8, Name: "u64", IsPrimitive: true}
	TypeInfoFloat32 = &TypeInfo{Size: 4, Alignment: 4, Name: "f32", IsPrimitive: true}
	TypeInfoFloat64 = &TypeInfo{Size: 8, Alignment: 8, Name: "f64", IsPrimitive: true}
	TypeInfoBool    = &TypeInfo{Size: 1, Alignment: 1, Name: "bool", IsPrimitive: true}
	TypeInfoChar    = &TypeInfo{Size: 4, Alignment: 4, Name: "char", IsPrimitive: true}
	TypeInfoPtr     = &TypeInfo{Size: 8, Alignment: 8, Name: "ptr", IsPointer: true}
)

// Utility functions.

func copyBytes(dest, src unsafe.Pointer, size uintptr) {
	if dest == nil || src == nil || size == 0 {
		return
	}
	// Simple byte copy implementation.
	destBytes := (*[1 << 30]byte)(dest)[:size:size]
	srcBytes := (*[1 << 30]byte)(src)[:size:size]
	copy(destBytes, srcBytes)
}

func compareBytes(ptr1, ptr2 unsafe.Pointer, size uintptr) int {
	bytes1 := (*[1 << 30]byte)(ptr1)[:size:size]
	bytes2 := (*[1 << 30]byte)(ptr2)[:size:size]

	for i := uintptr(0); i < size; i++ {
		if bytes1[i] < bytes2[i] {
			return -1
		} else if bytes1[i] > bytes2[i] {
			return 1
		}
	}

	return 0
}

func hashBytes(data []byte) uint64 {
	// Simple FNV-1a hash.
	const (
		fnvPrime  = 1099511628211
		fnvOffset = 14695981039346656037
	)

	hash := uint64(fnvOffset)
	for _, b := range data {
		hash ^= uint64(b)
		hash *= fnvPrime
	}

	return hash
}

func cStringLength(cstr unsafe.Pointer) uintptr {
	if cstr == nil {
		return 0
	}

	bytes := (*[1 << 30]byte)(cstr)
	length := uintptr(0)

	for bytes[length] != 0 {
		length++
	}

	return length
}
