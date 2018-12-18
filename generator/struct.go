package generator

import (
	"fmt"
	"github.com/dennwc/go-doxy"
	"github.com/dennwc/go-doxy/xmlfile"
	"github.com/dennwc/go-doxy/xmlindex"
	"io"
	"log"
	"strings"
)

var (
	_ attributeHost = (*StructType)(nil)
	_ propertyHost  = (*StructType)(nil)
	_ methodHost    = (*StructType)(nil)
)

type StructType struct {
	BaseNode
	IsClass    bool
	Attributes []*Attribute
	Properties []*Property
	Methods    []*Function
}

func (t *StructType) getName() string {
	return t.Name
}

func (t *StructType) addAttribute(f *Attribute) {
	t.Attributes = append(t.Attributes, f)
}

func (t *StructType) addProperty(f *Property) {
	t.Properties = append(t.Properties, f)
}

func (t *StructType) addMethod(f *Function) {
	for _, f2 := range t.Methods {
		if f2.Name == f.Name {
			log.Printf("redeclaration of %q.%q", t.Name, f.Name)
			return
		}
	}
	t.Methods = append(t.Methods, f)
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
	// no need to cast - implements objc.Object
	return exp, true
}

func (t *StructType) CastToGo(exp string) (string, bool) {
	if !t.ensureGoName() {
		return "", false
	}
	name, ok := t.GoTypeName()
	if !ok {
		return exp, false
	}
	return "As" + name + "(" + exp + ")", true
}

func toExportedName(s string) string {
	return toGoName(s, true)
}

func (t *StructType) goMethName(name string) string {
	if strings.HasSuffix(name, ":") {
		// TODO: only if collides?
		name = strings.TrimSuffix(name, ":") + "_"
	}
	name = toExportedName(strings.Replace(name, ":", "_", -1))
	return name
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
	return As%s(objc.GetClass(%q).SendMsg("alloc").SendMsg("init"))
}

func As%s(v objc.Object) %s {
	return %s{v}
}
`,
		t.GoName,

		t.GoName, t.GoName,
		t.GoName, t.Name,

		t.GoName, t.GoName,
		t.GoName,
	)

	// setters
	for _, p := range t.Properties {
		name := toExportedName(p.Name)
		tp, ok := p.Type.GoTypeName()
		if !ok {
			fmt.Fprintf(w, "\n// TODO: property %s (%#v)\n", name, p.Type)
			continue
		}
		cast, ok := p.Type.CastToObjC("v")
		if !ok {
			fmt.Fprintf(w, "\n// TODO: property %s (%#v)\n", name, p.Type)
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
					callStmt = "return " + cast + fmt.Sprintf(" // from %#v", ft.Return)
				} else {
					returnType = " /* TODO: " + typ + " */"
				}
			} else {
				comment = fmt.Sprintf("\n\t// FIXME: return %#v", ft.Return)
			}
		}

		name := t.goMethName(m.Name)
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

type attributeHost interface {
	TypeDefinition
	addAttribute(a *Attribute)
}

type Attribute struct {
	Name   string
	Prot   Protection
	Static bool

	Type Type

	Pos   *Location
	Range *LineRange
}

type propertyHost interface {
	TypeDefinition
	addProperty(p *Property)
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
	return g.loadDoxyObject(t, ent, def)
}

func (g *Generator) loadDoxyObject(t TypeDefinition, ent doxy.Entry, def xmlfile.CompounddefType) error {
	defAttr, _ := t.(attributeHost)
	defProp, _ := t.(propertyHost)
	defFnc, _ := t.(methodHost)
	for _, sec := range def.Sectiondef {
		switch sec.Kind {
		case "public-attrib", "public-static-attrib",
			"protected-attrib", "protected-static-attrib",
			"private-attrib", "private-static-attrib":
			if defAttr == nil {
				return fmt.Errorf("%T cannot host attributes", t)
			}
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

				defAttr.addAttribute(f)

				if attr.Kind != "variable" {
					log.Println("unexpected struct attribute kind:", attr.Kind)
				}
			}
		case "property":
			if defProp == nil {
				return fmt.Errorf("%T (%s) cannot host properties", t, t.getName())
			}
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

				defProp.addProperty(f)

				if attr.Kind != "property" {
					log.Println("unexpected struct property kind:", attr.Kind)
				}
			}
		case "func", "public-func": // TODO: static ("public-static-func")
			if defFnc == nil {
				return fmt.Errorf("%T cannot host methods", t)
			}
			if err := g.loadFuncs(defFnc, sec.Memberdef); err != nil {
				return err
			}
		default:
			//log.Println("unhandled object section:", sec.Kind)
		}
	}
	return nil
}

func (g *Generator) TypeByName(name string) TypeDefinition {
	for _, t := range g.types {
		tname, ok := t.GoTypeName()
		if !ok {
			continue
		}
		if tname == name {
			return t
		}
	}
	return nil
}
