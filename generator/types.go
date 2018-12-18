package generator

import (
	"io"
	"strings"
	"unicode"

	"github.com/dennwc/go-doxy"
	"github.com/dennwc/go-doxy/xmlfile"
)

type Protection string

const (
	Public = Protection("public")
)

type Type interface {
	// GoTypeName returns a code snippet of the Go type that corresponds to ObjC type.
	// If the returned bool is false, this means that the type cannot be converted yet.
	GoTypeName() (string, bool)
	// CastToObjC casts an expression from Go type returned by GoTypeName to ObjC type.
	CastToObjC(exp string) (string, bool)
	// CastToGo casts an expression from objc.Object to a Go type returned by GoTypeName.
	CastToGo(exp string) (string, bool)
}

type TypeDefinition interface {
	Type
	getName() string
	PrintGoWrapper(w io.Writer) bool
}

var nameReplacer = strings.NewReplacer(
	":", "_",
	"-", "_",
)

func toGoName(name string, exp bool) string {
	if name == "" {
		return ""
	}
	name = nameReplacer.Replace(name)
	if exp {
		name = string(unicode.ToUpper(rune(name[0]))) + name[1:]
	} else {
		name = string(unicode.ToLower(rune(name[0]))) + name[1:]
	}
	switch name {
	case "type":
		return "typ"
	case "select":
		return "sel"
	case "range":
		return "rng"
	}
	return name
}

type BaseNode struct {
	refid  string
	Name   string
	GoName string
	Prot   Protection

	Pos   *Location
	Range *LineRange
}

func (t *BaseNode) ensureGoName() bool {
	if t.Name == "" {
		return false
	}
	if t.GoName == "" {
		t.GoName = nameReplacer.Replace(t.Name)
	}
	return true
}

func entToBaseNode(ent doxy.Entry, def xmlfile.CompounddefType) BaseNode {
	return BaseNode{
		refid: ent.Refid,
		Name:  ent.Name,
		Prot:  Protection(def.Prot),
		Pos:   asLocation(def.Location),
		Range: asLineRange(def.Location),
	}
}
