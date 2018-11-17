package objc

/*
#cgo LDFLAGS: -Wl,--no-as-needed -lobjc
#define __OBJC2__ 1
#include <objc/runtime.h>
#include <objc/message.h>
*/
import "C"

import "unsafe"

func GetClass(name string) *Class {
	cstr := C.CString(name)
	c := C.objc_getClass(cstr)
	freeString(cstr)
	if c == nil {
		return nil
	}
	return &Class{class: c}
}

func ListClasses() []Class {
	n := int(C.objc_getClassList(nil, 0))
	if n == 0 {
		return nil
	}
	var cl C.Class
	const sz = unsafe.Sizeof(cl)

	buf := malloc(uintptr(n) * sz)
	_ = C.objc_getClassList((*C.Class)(buf), C.int(n))

	out := make([]Class, 0, n)
	for i := 0; i < n; i++ {
		off := uintptr(i) * sz
		c := (*C.Class)(incPtr(buf, off))
		out = append(out, Class{class: *c})
	}
	free(buf)
	return out
}

type Class struct {
	class C.Class
}

func (c *Class) Valid() bool {
	return c != nil && c.class != nil
}

func (c Class) String() string {
	if !c.Valid() {
		return "<nil>"
	}
	return c.Name()
}

func (c Class) Name() string {
	if !c.Valid() {
		return ""
	}
	cstr := C.class_getName(c.class)
	return C.GoString(cstr)
}

func (c *Class) GetSuperclass() *Class {
	if c == nil {
		return nil
	}
	s := C.class_getSuperclass(c.class)
	if s == nil {
		return nil
	}
	return &Class{class: s}
}

func (c *Class) IsMetaClass() bool {
	if c == nil {
		return false
	}
	v := C.class_isMetaClass(c.class)
	return v != 0
}

func (c *Class) GetInstanceSize() uintptr {
	if c == nil {
		return 0
	}
	v := C.class_getInstanceSize(c.class)
	return uintptr(v)
}
