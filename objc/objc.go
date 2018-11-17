package objc

import "C"

import "unsafe"

// GetClass returns the class definition of a specified class.
//
// See https://developer.apple.com/documentation/objectivec/1418952-objc_getclass?language=objc
func GetClass(name string) *Class {
	cstr := C.CString(name)
	c := objc_getClass(cstr)
	freeString(cstr)
	if c == nil {
		return nil
	}
	return &Class{class: c}
}

// ListClasses obtains the list of registered class definitions.
//
// See https://developer.apple.com/documentation/objectivec/1418579-objc_getclasslist?language=objc
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

// Name returns the name of a class.
//
// See https://developer.apple.com/documentation/objectivec/1418635-class_getname?language=objc
func (c Class) Name() string {
	if !c.Valid() {
		return ""
	}
	cstr := class_getName(c.class)
	return C.GoString(cstr)
}

// GetSuperclass returns the superclass of a class.
//
// See https://developer.apple.com/documentation/objectivec/1418498-class_getsuperclass?language=objc
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

// IsMetaClass returns a boolean value that indicates whether a class object is a metaclass.
//
// See https://developer.apple.com/documentation/objectivec/1418627-class_ismetaclass?language=objc
func (c *Class) IsMetaClass() bool {
	if c == nil {
		return false
	}
	return class_isMetaClass(c.class)
}

// GetInstanceSize returns the size of instances of a class.
//
// See https://developer.apple.com/documentation/objectivec/1418907-class_getinstancesize?language=objc
func (c *Class) GetInstanceSize() uintptr {
	if c == nil {
		return 0
	}
	return class_getInstanceSize(c.class)
}
