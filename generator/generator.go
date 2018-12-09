package generator

import (
	"io"
	"strings"

	"github.com/dennwc/go-doxy"
	"github.com/dennwc/go-doxy/xmlfile"
	"github.com/dennwc/go-doxy/xmlindex"
)

func NewGenerator() *Generator {
	return &Generator{
		types: make(map[refid]TypeDefinition),
		files: make(map[refid]BaseNode),
		funcs: make(map[string]*Function),
	}
}

type refid = string

type Generator struct {
	types map[refid]TypeDefinition
	files map[refid]BaseNode
	funcs map[string]*Function
}

func (g *Generator) LoadDoxygen(dirs ...string) error {
	for _, dir := range dirs {
		idx, err := doxy.OpenXML(dir)
		if err != nil {
			return err
		}
		for _, ent := range idx.Entries() {
			switch ent.Kind {
			case xmlindex.CompoundKindStruct,
				xmlindex.CompoundKindClass,
				xmlindex.CompoundKindInterface:
				if err := g.loadDoxyStruct(ent); err != nil {
					return err
				}
			case xmlindex.CompoundKindFile:
				if err := g.loadDoxyFile(ent); err != nil {
					return err
				}
			case xmlindex.CompoundKindProtocol:
				if err := g.loadDoxyProtocol(ent); err != nil {
					return err
				}
			default:
				//log.Println("unhandled entry type:", ent.Kind)
			}
		}
	}
	return nil
}

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
		t.GoName = strings.Replace(t.Name, ":", "_", -1)
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
