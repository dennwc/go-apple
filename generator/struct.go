package generator

import (
	"fmt"
	"io"
	"log"
	"strings"
	"unicode"

	"github.com/dennwc/go-doxy"
	"github.com/dennwc/go-doxy/xmlindex"
)

type StructType struct {
	BaseNode
	IsClass    bool
	Attributes []*Attribute
	Properties []*Property
	Methods    []*Function
}

func (t *StructType) GoTypeName() (string, bool) {
	if !t.ensureGoName() {
		return "", false
	}
	return t.GoName, true
}

func (t *StructType) CastToObjC(exp string) (string, bool) {
	if !t.ensureGoName() {
		return "", false
	}
	return exp, true
}

func (t *StructType) CastToGo(exp string) (string, bool) {
	if !t.ensureGoName() {
		return "", false
	}
	return exp, true
}

func toExportedName(s string) string {
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func (t *StructType) PrintGoWrapper(w io.Writer) bool {
	if !t.ensureGoName() {
		return false
	}
	fmt.Fprintf(w, "// %s", t.Name)
	if p := t.Pos; p != nil {
		fmt.Fprintf(w, " (%s)", p)
	}
	fmt.Fprintf(w, `
type %s struct {
	objc.Object
}

func New%s() %s {
	return %s{objc.GetClass(%q).SendMsg("alloc").SendMsg("init")}
}
`,
		t.GoName,
		t.GoName, t.GoName,
		t.GoName, t.Name,
	)

	// setters
	for _, p := range t.Properties {
		name := toExportedName(p.Name)
		tp, ok := p.Type.GoTypeName()
		if !ok {
			log.Printf("skipping %q - %#v", name, p.Type)
			continue
		}
		cast, ok := p.Type.CastToObjC("v")
		if !ok {
			log.Printf("skipping %q - %#v", name, p.Type)
			continue
		}
		fmt.Fprintf(w, `
func (o %s) Set%s(v %s) {
	o.SendMsg(%q, %s)
}
`,
			t.GoName, name, tp,
			"set"+name+":", cast,
		)
	}
	// methods
methods:
	for _, m := range t.Methods {
		ft := m.Type
		for _, p := range ft.Args {
			_, ok := p.Type.GoTypeName()
			if !ok {
				continue methods
			}
		}

		callStmt := fmt.Sprintf("o.SendMsg(%q", m.Name)
		for _, p := range ft.Args {
			cast, ok := p.Type.CastToObjC(p.Name)
			if !ok {
				continue methods
			}
			callStmt += ", " + cast
		}
		callStmt += ")"

		comment := ""

		returnType := ""
		if ft.Return != nil {
			// let's dump the method anyway without return if this fails
			if typ, ok := ft.Return.GoTypeName(); ok {
				cast, ok := ft.Return.CastToGo(callStmt)
				if ok {
					returnType = " " + typ
					callStmt = "return " + cast
				} else {
					returnType = " /* TODO: " + typ + " */"
				}
			} else {
				comment = fmt.Sprintf("\n\t// FIXME: return %#v", ft.Return)
			}
		}

		name := toExportedName(strings.TrimSuffix(m.Name, ":"))
		fmt.Fprintf(w, `
func (o %s) %s(`,
			t.GoName, name,
		)
		for i, p := range m.Type.Args {
			if i != 0 {
				fmt.Fprint(w, ", ")
			}
			typ, _ := p.Type.GoTypeName()
			fmt.Fprintf(w, `%s %s`, p.Name, typ)
		}
		fmt.Fprintf(w, ")%s {\n\t%s%s\n}\n", returnType, callStmt, comment)
	}
	return true
}

func (t *StructType) printGoDef(w io.Writer) bool {
	if !t.ensureGoName() {
		return false
	}
	fmt.Fprintf(w, "// %s", t.Name)
	if p := t.Pos; p != nil {
		fmt.Fprintf(w, " (%s)", p)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "type %s struct {\n", t.GoName)
	ok := true
	for _, f := range t.Attributes {
		ft, ok2 := f.Type.GoTypeName()
		if !ok2 {
			ok = false
		}
		fmt.Fprintf(w, "\t%s %s", f.Name, ft)
		if p := f.Pos; p != nil {
			fmt.Fprintf(w, "\t// %s", p)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprint(w, "}\n\n")
	return ok
}

type Attribute struct {
	Name   string
	Prot   Protection
	Static bool

	Type Type

	Pos   *Location
	Range *LineRange
}

type Property struct {
	Name     string
	Type     Type
	Readable bool
	Writable bool

	Pos   *Location
	Range *LineRange
}

func (g *Generator) loadDoxyStruct(ent doxy.Entry) error {
	//log.Printf("struct: %q", ent.Name)
	typ, err := ent.Decode()
	if err != nil {
		return err
	}
	def := typ.Compounddef
	_ = def.Language // TODO: Objective-C, C++

	t, _ := g.types[ent.Refid].(*StructType)
	if t == nil {
		t = &StructType{
			BaseNode: BaseNode{refid: ent.Refid},
		}
		g.types[t.refid] = t
	}
	*t = StructType{
		BaseNode: entToBaseNode(ent, def),
		IsClass:  ent.Kind == xmlindex.CompoundKindClass,
	}

	for _, sec := range def.Sectiondef {
		switch sec.Kind {
		case "public-attrib", "public-static-attrib",
			"protected-attrib", "protected-static-attrib",
			"private-attrib", "private-static-attrib":
			for _, attr := range sec.Memberdef {
				f := &Attribute{
					Name:   attr.Name,
					Static: strings.Contains(string(sec.Kind), "static"),
					Prot:   Protection(attr.Prot),
					Pos:    asLocation(attr.Location),
					Range:  asLineRange(attr.Location),
				}

				typ, err := g.getMemberType(attr)
				if err != nil {
					return err
				}
				f.Type = typ

				t.Attributes = append(t.Attributes, f)

				if attr.Kind != "variable" {
					log.Println("unexpected struct attribute kind:", attr.Kind)
				}
			}
		case "property":
			for _, attr := range sec.Memberdef {
				f := &Property{
					Name:     attr.Name,
					Pos:      asLocation(attr.Location),
					Range:    asLineRange(attr.Location),
					Readable: bool(attr.Readable),
					Writable: bool(attr.Writable),
				}

				typ, err := g.getMemberType(attr)
				if err != nil {
					return err
				}
				f.Type = typ

				t.Properties = append(t.Properties, f)

				if attr.Kind != "property" {
					log.Println("unexpected struct property kind:", attr.Kind)
				}
			}
		case "func", "public-func":
			if err := g.loadFuncs(sec.Memberdef); err != nil {
				return err
			}
		default:
			//log.Println("unhandled struct section:", sec.Kind)
		}
	}
	return nil
}

func (g *Generator) StructByName(name string) *StructType {
	for _, t := range g.types {
		s, ok := t.(*StructType)
		if !ok {
			continue
		}
		if s.Name == name {
			return s
		}
	}
	return nil
}
