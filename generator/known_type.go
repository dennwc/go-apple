package generator

type NSString struct{}

func (NSString) GoTypeName() (string, bool) {
	return "string", true
}

func (NSString) CastToObjC(exp string) (string, bool) {
	return "foundation.NSStringFromString(" + exp + ")", true
}

func (NSString) CastToGo(exp string) (string, bool) {
	return exp + ".String()", true
}

type NSNotification struct{}

func (NSNotification) GoTypeName() (string, bool) {
	// TODO: define it
	return "objc.Object", true
}

func (NSNotification) CastToObjC(exp string) (string, bool) {
	return exp, true
}

func (NSNotification) CastToGo(exp string) (string, bool) {
	return exp, true
}
