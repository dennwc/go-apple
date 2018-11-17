package objc

import "C"

import "unsafe"

func GetClass(name string) *Class {
	cstr := C.CString(name)
	c := objc_getClass(cstr)
	freeString(cstr)
	if c == nil {
		return nil
	}
	return &Class{class: c}
}

func ListClasses() []Class {
	n := objc_getClassList(nil, 0)
	if n == 0 {
		return nil
	}
	var cl cClass
	const sz = unsafe.Sizeof(cl)

	buf := malloc(uintptr(n) * sz)
	_ = objc_getClassList((*cClass)(buf), n)

	out := make([]Class, 0, n)
	for i := 0; i < n; i++ {
		off := uintptr(i) * sz
		c := (*cClass)(incPtr(buf, off))
		out = append(out, Class{class: *c})
	}
	free(buf)
	return out
}

type Class struct {
	class cClass
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
	cstr := class_getName(c.class)
	return C.GoString(cstr)
}

func (c *Class) GetSuperclass() *Class {
	if c == nil {
		return nil
	}
	s := class_getSuperclass(c.class)
	if s == nil {
		return nil
	}
	return &Class{class: s}
}

func (c *Class) IsMetaClass() bool {
	if c == nil {
		return false
	}
	return class_isMetaClass(c.class)
}

func (c *Class) GetInstanceSize() uintptr {
	if c == nil {
		return 0
	}
	return class_getInstanceSize(c.class)
}
