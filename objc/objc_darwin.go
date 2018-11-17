package objc

/*
#cgo LDFLAGS: -lobjc
#include <objc/runtime.h>
#include <objc/message.h>
*/
import "C"

type cClass = C.Class

func objc_getClass(name *C.char) cClass {
	return C.objc_getClass(name)
}

func objc_getClassList(buf *cClass, n int) int {
	return int(C.objc_getClassList(buf, C.int(n)))
}

func class_getName(c cClass) *C.char {
	return C.class_getName(c)
}

func class_getSuperclass(c cClass) cClass {
	return C.class_getSuperclass(c)
}

func class_isMetaClass(c cClass) bool {
	return C.class_isMetaClass(c) != 0
}

func class_getInstanceSize(c cClass) uintptr {
	return uintptr(C.class_getInstanceSize(c))
}
