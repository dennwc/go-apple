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
