package generator

import (
	"fmt"
	"io"
	"strings"

	"github.com/dennwc/go-doxy"
)

var (
	_ methodHost = (*ProtocolType)(nil)
)

type ProtocolType struct {
	BaseNode
	Properties []*Property
	Methods    []*Function
}

func (t *ProtocolType) getName() string {
	return t.Name
}

func (t *ProtocolType) addMethod(f *Function) {
	t.Methods = append(t.Methods, f)
}

func (t *ProtocolType) addProperty(f *Property) {
	t.Properties = append(t.Properties, f)
}

func (t *ProtocolType) GoTypeName() (string, bool) {
	if !t.ensureGoName() {
		return "", false
	}
	return t.GoName, true
}

func (t *ProtocolType) CastToObjC(exp string) (string, bool) {
	if !t.ensureGoName() {
		return "", false
	}
	return exp, true
}

func (t *ProtocolType) CastToGo(exp string) (string, bool) {
	if !t.ensureGoName() {
		return "", false
	}
	return exp, true
}

func (t *ProtocolType) PrintGoWrapper(w io.Writer) bool {
	if !t.ensureGoName() {
		return false
	}
	if !t.printGoInterface(w) {
		return false
	}
	if !t.printGoImpl(w) {
		return false
	}
	return true
}

func (t *ProtocolType) printGoInterface(w io.Writer) bool {
	fmt.Fprintf(w, "// %s", t.Name)
	if p := t.Pos; p != nil {
		fmt.Fprintf(w, " (%s)", p)
	}
	// generate Go interface that user needs to implement
	fmt.Fprintf(w, `
type %s interface {
	objc.Object
	SetObjcRef(v objc.Object)

`,
		t.GoName,
	)
	// methods
methods:
	for _, m := range t.Methods {
		if !m.ensureGoName() {
			continue
		}
		ft := m.Type
		for _, p := range ft.Args {
			_, ok := p.Type.GoTypeName()
			if !ok {
				fmt.Fprintf(w, "\n\t// TODO: %s (%#v)\n\n", m.Name, p.Type)
				continue methods
			}
			_, ok = p.Type.CastToObjC("v")
			if !ok {
				fmt.Fprintf(w, "\n\t// TODO: %s (%#v)\n\n", m.Name, p.Type)
				continue methods
			}
		}

		comment := ""

		returnType := ""
		if ft.Return != nil {
			// let's dump the method anyway without return if this fails
			if typ, ok := ft.Return.GoTypeName(); ok {
				_, ok := ft.Return.CastToGo("v")
				if ok {
					returnType = " " + typ
				} else {
					returnType = " /* TODO: " + typ + " */"
				}
			} else {
				comment = fmt.Sprintf("\t// FIXME: return %#v\n", ft.Return)
			}
		}

		name := toExportedName(strings.Replace(strings.TrimSuffix(m.Name, ":"), ":", "_", -1))
		fmt.Fprintf(w, "%s\t%s(",
			comment,
			name,
		)
		for i, p := range m.Type.Args {
			if i != 0 {
				fmt.Fprint(w, ", ")
			}
			typ, _ := p.Type.GoTypeName()
			fmt.Fprintf(w, `%s %s`, p.Name, typ)
		}
		fmt.Fprintf(w, ")%s\n", returnType)
	}
	fmt.Fprint(w, "}\n")
	return true
}

func (t *ProtocolType) printGoImpl(w io.Writer) bool {
	// Go wrapper type
	fmt.Fprintf(w,
		"\ntype go%s struct{\n\tobjc.Object `objc:\"go%s : %s\"`\n\tv %s\n}\n",
		t.GoName, t.GoName, t.Name, t.GoName,
	)
methods:
	for _, m := range t.Methods {
		if !m.ensureGoName() {
			continue
		}
		ft := m.Type
		for _, p := range ft.Args {
			_, ok := p.Type.GoTypeName()
			if !ok {
				continue methods
			}
			_, ok = p.Type.CastToObjC("v")
			if !ok {
				continue methods
			}
		}

		returnType := ""
		if ft.Return != nil {
			if typ, ok := ft.Return.GoTypeName(); ok {
				_, ok := ft.Return.CastToGo("v")
				if ok {
					returnType = " " + typ
				}
			}
		}

		name := toExportedName(strings.Replace(strings.TrimSuffix(m.Name, ":"), ":", "_", -1))
		fmt.Fprintf(w, "func (o go%s) %s(",
			t.GoName,
			name,
		)
		for i, p := range m.Type.Args {
			if i != 0 {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprintf(w, `%s objc.Object`, p.Name)
		}
		fmt.Fprint(w, ")")
		if returnType != "" {
			fmt.Fprint(w, " objc.Object")
		}
		fmt.Fprint(w, " {\n\t")
		callStmt := fmt.Sprintf("o.v.%s(", name)
		for i, p := range m.Type.Args {
			if i != 0 {
				callStmt += ", "
			}
			cast, _ := p.Type.CastToGo(p.Name)
			callStmt += cast
		}
		callStmt += ")"
		if returnType != "" {
			callStmt, _ = ft.Return.CastToObjC(callStmt)
			callStmt = "return " + callStmt
		}
		fmt.Fprint(w, callStmt)
		fmt.Fprint(w, "\n}\n")
	}
	if !t.printGoClassReg(w) {
		return false
	}
	return true
}

func (t *ProtocolType) printGoClassReg(w io.Writer) bool {
	// register class methods
	fmt.Fprintf(w, `
func init(){
	c := objc.NewClass(go%s{})`,
		t.GoName,
	)
methods:
	for _, m := range t.Methods {
		if !m.ensureGoName() {
			continue
		}
		ft := m.Type
		for _, p := range ft.Args {
			_, ok := p.Type.GoTypeName()
			if !ok {
				continue methods
			}
			_, ok = p.Type.CastToObjC("v")
			if !ok {
				continue methods
			}
		}

		name := toExportedName(strings.Replace(strings.TrimSuffix(m.Name, ":"), ":", "_", -1))
		fmt.Fprintf(w, `
	c.AddMethod(%q, go%s.%s)`,
			m.Name, t.GoName, name,
		)
	}
	fmt.Fprint(w, "\n\tobjc.RegisterClass(c)\n}\n")
	return true
}

func (g *Generator) loadDoxyProtocol(ent doxy.Entry) error {
	//log.Printf("protocol: %q", ent.Name)
	typ, err := ent.Decode()
	if err != nil {
		return err
	}
	def := typ.Compounddef
	_ = def.Language // TODO: Objective-C, C++

	t, _ := g.types[ent.Refid].(*ProtocolType)
	if t == nil {
		t = &ProtocolType{
			BaseNode: BaseNode{refid: ent.Refid},
		}
		g.types[t.refid] = t
	}
	*t = ProtocolType{
		BaseNode: entToBaseNode(ent, def),
	}
	t.Name = strings.TrimSuffix(t.Name, " -p")
	return g.loadDoxyObject(t, ent, def)
}
