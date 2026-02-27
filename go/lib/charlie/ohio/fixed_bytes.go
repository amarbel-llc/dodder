package ohio

import "unsafe"

// TODO write tests

func Int64ToByteArray(i int64) [8]byte {
	return *(*[unsafe.Sizeof(i)]byte)(unsafe.Pointer(&i))
}

func ByteArrayToUInt8(arr [1]byte) uint8 {
	val := uint8(0)
	size := len(arr)

	for i := range size {
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(i))) = arr[i]
	}

	return val
}

func ByteArrayToInt8(arr [1]byte) int8 {
	val := int8(0)
	size := len(arr)

	for i := range size {
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(i))) = arr[i]
	}

	return val
}

func ByteArrayToInt16(arr [2]byte) int16 {
	val := int16(0)
	size := len(arr)

	for i := range size {
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(i))) = arr[i]
	}

	return val
}

func ByteArrayToUInt16(arr [2]byte) uint16 {
	val := uint16(0)
	size := len(arr)

	for i := range size {
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(i))) = arr[i]
	}

	return val
}

func ByteArrayToInt64(arr [8]byte) int64 {
	val := int64(0)
	size := len(arr)

	for i := range size {
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(i))) = arr[i]
	}

	return val
}

func UInt8ToByteArray(i uint8) [1]byte {
	return *(*[unsafe.Sizeof(i)]byte)(unsafe.Pointer(&i))
}

func Int8ToByteArray(i int8) [1]byte {
	return *(*[unsafe.Sizeof(i)]byte)(unsafe.Pointer(&i))
}

func Int16ToByteArray(i int16) [2]byte {
	return *(*[unsafe.Sizeof(i)]byte)(unsafe.Pointer(&i))
}

func UInt16ToByteArray(i uint16) [2]byte {
	return *(*[unsafe.Sizeof(i)]byte)(unsafe.Pointer(&i))
}

func Int32ToByteArray(i int32) [4]byte {
	return *(*[unsafe.Sizeof(i)]byte)(unsafe.Pointer(&i))
}

func ByteArrayToInt32(arr [4]byte) int32 {
	val := int32(0)
	size := len(arr)

	for i := range size {
		*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(&val)) + uintptr(i))) = arr[i]
	}

	return val
}
