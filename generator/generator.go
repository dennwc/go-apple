package generator

import (
	"bytes"
	"io"
	"sort"
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

func (g *Generator) LoadDoxygen(dir string) error {
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
		//case xmlindex.CompoundKindUnion:
		// TODO
		case xmlindex.CompoundKindFile:
			if err := g.loadDoxyFile(ent); err != nil {
				return err
			}
		default:
			//log.Println("unhandled entry type:", ent.Kind)
		}
	}
	if err := g.attachMethods(); err != nil {
		return err
	}
	return nil
}

func (g *Generator) PrintGo(w io.Writer) {
	refs := make([]string, 0, len(g.types))
	//for ref := range g.types {
	//	refs = append(refs, ref)
	//}
	//sort.Strings(refs)
	//for _, ref := range refs {
	//	buf := bytes.NewBuffer(nil)
	//	if g.types[ref].printGoDef(buf) {
	//		buf.WriteTo(w)
	//	}
	//}

	refs = refs[:0]
	for name := range g.funcs {
		refs = append(refs, name)
	}
	sort.Strings(refs)
	for _, name := range refs {
		buf := bytes.NewBuffer(nil)
		if g.funcs[name].printGoDef(buf) {
			buf.WriteTo(w)
		}
	}
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
}

type TypeDefinition interface {
	Type
	printGoDef(w io.Writer) bool
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
