package generator

import (
	"fmt"
	"log"
	"strings"

	"github.com/dennwc/go-doxy"
	"github.com/dennwc/go-doxy/xmlfile"
)

type Function struct {
	BaseNode
	Type     *FuncType
	Extern   bool
	receiver string
}

func (g *Generator) loadDoxyFile(ent doxy.Entry) error {
	//log.Printf("file: %q", ent.Name)
	typ, err := ent.Decode()
	if err != nil {
		return err
	}
	def := typ.Compounddef
	_ = def.Language // TODO: Objective-C, C++

	if _, ok := g.files[ent.Refid]; ok {
		return fmt.Errorf("duplicated file definition")
	}
	g.files[ent.Refid] = entToBaseNode(ent, def)

	for _, sec := range def.Sectiondef {
		switch sec.Kind {
		case "func", "public-func":
			if err := g.loadFuncs(nil, sec.Memberdef); err != nil {
				return err
			}
		default:
			//log.Println("unhandled file section:", sec.Kind)
		}
	}
	return nil
}

type methodHost interface {
	TypeDefinition
	addMethod(f *Function)
}

func (g *Generator) loadFuncs(rec methodHost, funcs []xmlfile.MemberdefType) error {
	for _, m := range funcs {
		if m.Kind != "function" {
			log.Println("unhandled function kind:", m.Kind)
			continue
		}
		f := &Function{
			BaseNode: BaseNode{
				Name:  m.Name,
				Prot:  Protection(m.Prot),
				Pos:   asLocation(m.Location),
				Range: asLineRange(m.Location),
			},
			Type: &FuncType{},
		}
		ft := f.Type

		if i := strings.Index(m.Definition, "::"); i >= 0 {
			recv := m.Definition[:i]
			recv = strings.TrimSuffix(recv, " -p") // for protocols
			i = strings.LastIndex(recv, " ")
			recv = recv[i+1:]
			f.receiver = recv
		}
		if rec != nil && f.receiver != "" && rec.getName() == f.receiver {
			rec.addMethod(f)
		} else {
			g.funcs[f.receiver+"::"+f.Name] = f
		}

		ret, err := g.getTypeFromLinkText(m.Type)
		if err != nil {
			return err
		}
		if e, ok := ret.(appkitExtern); ok {
			ret = e.Type
			f.Extern = true
		}
		ft.Return = ret

		for _, p := range m.Param {
			arg := &FuncArg{
				Name: p.Declname,
			}
			ft.Args = append(ft.Args, arg)

			at, err := g.getTypeFromLinkText(p.Type)
			if err != nil {
				return err
			}
			if p.Array != "" {
				at = ArrayType{Elem: at} // TODO
			}
			arg.Type = at
		}
	}
	return nil
}

func (g *Generator) FunctionByName(name string) *Function {
	return g.funcs[name]
}
