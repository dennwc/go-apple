package objc

// #include <stdlib.h>
import "C"

import "unsafe"

func free(p unsafe.Pointer) {
	C.free(p)
}

func freeString(s *C.char) {
	free(unsafe.Pointer(s))
}

func malloc(sz uintptr) unsafe.Pointer {
	return C.malloc(C.size_t(sz))
}

func incPtr(p unsafe.Pointer, i uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + i)
}
