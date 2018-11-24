package generator

import (
	"fmt"
	"io"
	"log"
	"strings"
	"unicode"

	"github.com/dennwc/go-doxy/xmlfile"
)

var (
	overrideTypes = map[string]Type{
		"": PrimitiveType{Name: "interface{}"},
		// TODO: unsafe.Pointer?
		"void *":          PrimitiveType{Name: "uintptr"},
		"__strong void *": PrimitiveType{Name: "uintptr"},               // TODO
		"char *":          ArrayType{Elem: PrimitiveType{Name: "byte"}}, // []byte
	}
	primitiveTypes = map[string]string{
		"bool":               "bool",
		"char":               "byte",
		"int8_t":             "int8",
		"int16_t":            "int16",
		"int32_t":            "int32",
		"int64_t":            "int64",
		"uint8_t":            "uint8",
		"uint16_t":           "uint16",
		"uint32_t":           "uint32",
		"uint64_t":           "uint64",
		"int":                "int",     // TODO
		"long":               "int64",   // TODO
		"long long":          "int64",   // TODO
		"unsigned":           "uint",    // TODO
		"unsigned int":       "uint",    // TODO
		"unsigned short":     "uint16",  // TODO
		"signed int":         "int",     // TODO
		"unsigned long":      "uint64",  // TODO
		"unsigned long long": "uint64",  // TODO
		"uintptr_t":          "uintptr", // TODO: unsafe.Pointer?
	}
)

type PrimitiveType struct {
	Name string
}

func (t PrimitiveType) printGo(w io.Writer) bool {
	fmt.Fprint(w, t.Name)
	return true
}

type NamedType struct {
	Name string
}

func (t NamedType) printGo(w io.Writer) bool {
	fmt.Fprint(w, t.Name)
	return true
}

type ConstType struct {
	Elem Type
}

func (t ConstType) printGo(w io.Writer) bool {
	return t.Elem.printGo(w)
}

type StrongType struct {
	Elem Type
}

func (t StrongType) printGo(w io.Writer) bool {
	return t.Elem.printGo(w)
}

type NullableType struct {
	Elem     Type
	Nullable bool
}

func (t NullableType) printGo(w io.Writer) bool {
	return t.Elem.printGo(w)
}

type appkitExtern struct {
	Elem Type
}

func (t appkitExtern) printGo(w io.Writer) bool {
	return t.Elem.printGo(w)
}

type PtrType struct {
	Elem Type
}

func (t PtrType) printGo(w io.Writer) bool {
	fmt.Fprint(w, "*")
	return t.Elem.printGo(w)
}

type ArrayType struct {
	Size string // TODO: expression
	Elem Type
}

func (t ArrayType) printGo(w io.Writer) bool {
	fmt.Fprintf(w, "[%s]", t.Size)
	return t.Elem.printGo(w)
}

type FuncType struct {
	Return Type
	Args   []*FuncArg
}

type FuncArg struct {
	Name string
	Type Type
}

func (t *FuncType) printGo(w io.Writer) bool {
	fmt.Fprint(w, "func")
	return t.printGoArgs(w)
}
func (t *FuncType) printGoArgs(w io.Writer) bool {
	fmt.Fprint(w, "(")
	hasNames := false
	for _, a := range t.Args {
		if a.Name != "" {
			hasNames = true
			break
		}
	}
	ok := true
	for i, a := range t.Args {
		if i != 0 {
			fmt.Fprint(w, ", ")
		}
		name := a.Name
		if hasNames && name == "" {
			name = fmt.Sprintf("arg%d", i)
		}
		if name != "" {
			fmt.Fprintf(w, "%s ", name)
		}
		if a.Type == nil || !a.Type.printGo(w) {
			ok = false
			continue
		}
	}
	fmt.Fprint(w, ")")
	if t.Return != nil {
		fmt.Fprint(w, " ")
		if !t.Return.printGo(w) {
			ok = false
		}
	}
	return ok
}

type UnknownType struct {
	Name    string
	Comment string
}

func (t UnknownType) printGo(w io.Writer) bool {
	typ := "interface{}"
	if t.Name != "" {
		typ = t.Name
	}
	fmt.Fprint(w, typ)
	if t.Comment != "" {
		fmt.Fprintf(w, " /* %s */", t.Comment)
	}
	return false
}

var prefixWrappers = []struct {
	prefix string
	suffix string
	wrap   func(Type) Type
}{
	{prefix: "APPKIT_EXTERN ", wrap: func(e Type) Type {
		return appkitExtern{Elem: e}
	}},
	{prefix: "__nullable ", wrap: func(e Type) Type {
		return NullableType{Elem: e, Nullable: true}
	}},
	{prefix: "__null_unspecified ", wrap: func(e Type) Type {
		return e // TODO
	}},
	{prefix: "const ", wrap: func(e Type) Type {
		return ConstType{Elem: e}
	}},
	{prefix: "__strong ", wrap: func(e Type) Type {
		return StrongType{Elem: e}
	}},
	{suffix: "*", wrap: func(e Type) Type {
		return PtrType{Elem: e}
	}},
	{suffix: "_Nullable", wrap: func(e Type) Type {
		return NullableType{Elem: e, Nullable: true}
	}},
	{suffix: "__nonnull", wrap: func(e Type) Type {
		return NullableType{Elem: e, Nullable: false}
	}},
	{suffix: "__nullable", wrap: func(e Type) Type {
		return NullableType{Elem: e, Nullable: true}
	}},
	{suffix: "__null_unspecified", wrap: func(e Type) Type {
		return e // TODO
	}},
}

func (g *Generator) getTypeByName(typ string) (Type, error) {
	if typ == "void" {
		return nil, nil
	}
	orig := typ
	if t, ok := overrideTypes[typ]; ok {
		return t, nil
	}
	if strings.HasSuffix(typ, ")") {
		if i := strings.Index(typ, "(* )("); i >= 0 {
			ret := typ[:i]
			args := strings.Split(typ[i+len("(* )("):len(typ)-1], ", ")
			//log.Printf("%q -> %q %q", orig, ret, args)

			rt, err := g.getTypeByName(ret)
			if err != nil {
				return nil, err
			}
			fnc := &FuncType{Return: rt}

			for _, a := range args {
				suff := ""
				if strings.HasSuffix(a, "[]") {
					a = strings.TrimSuffix(a, "[]")
					suff = "[]"
				}
				li := -1
				for i := len(a) - 1; i >= 0; i-- {
					if c := rune(a[i]); !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' {
						li = i
						break
					}
				}
				var name string
				if li >= 0 && li < len(a)-1 {
					name = a[li+1:]
					a = strings.TrimSpace(a[:li+1])
				}
				a += suff
				//log.Printf("%q %q", name, a)
				typ, err := g.getTypeByName(a)
				if err != nil {
					return nil, err
				}
				fnc.Args = append(fnc.Args, &FuncArg{
					Name: name, Type: typ,
				})
			}
			return fnc, nil
		}
	}

	wrapWithPrefix := func(typ string, pref, suf string, wrap func(Type) Type) (Type, bool, error) {
		if pref != "" {
			if !strings.HasPrefix(typ, pref) {
				return nil, false, nil
			}
			typ = strings.TrimPrefix(typ, pref)
		} else {
			if !strings.HasSuffix(typ, suf) {
				return nil, false, nil
			}
			typ = strings.TrimSuffix(typ, suf)
		}
		typ = strings.TrimSpace(typ)
		elem, err := g.getTypeByName(typ)
		if err != nil {
			return nil, false, fmt.Errorf("%s(%q): %v", pref+suf, orig, err)
		}
		//if elem == nil {
		//	return nil, false, fmt.Errorf("nil type for %s element (%q)", pref+suf, orig)
		//}
		return wrap(elem), true, nil
	}

	for _, p := range prefixWrappers {
		if t, ok, err := wrapWithPrefix(typ, p.prefix, p.suffix, p.wrap); err != nil {
			return nil, err
		} else if ok {
			return t, nil
		}
	}
	if strings.HasSuffix(typ, "]") {
		if i := strings.LastIndex(typ, "["); i >= 0 {
			size := typ[i+1 : len(typ)-1]
			typ = strings.TrimSpace(typ[:i])
			elem, err := g.getTypeByName(typ)
			if err != nil {
				return nil, fmt.Errorf("arr(%q): %v", orig, err)
			}
			return ArrayType{Size: size, Elem: elem}, nil
		}
	}
	if t, ok := primitiveTypes[typ]; ok {
		return PrimitiveType{Name: t}, nil
	}
	if !strings.ContainsAny(typ, " (:") {
		return NamedType{Name: typ}, nil
	}
	return UnknownType{Comment: typ}, nil
}

func (g *Generator) getTypeFromLinkText(ft *xmlfile.LinkedTextType) (Type, error) {
	if ft == nil {
		return UnknownType{}, nil
	} else if len(ft.Ref) == 1 {
		ref := ft.Ref[0].Refid
		typ := g.types[ref]
		if typ == nil {
			typ = &StructType{BaseNode: BaseNode{refid: ref}} // TODO: UnresolvedType
			g.types[ref] = typ
		}
		return typ, nil
	} else if len(ft.Ref) != 0 {
		// FIXME: need to read XML tokens here - refs are mixed with text
		log.Printf("WARN: ref type: %+v", ft.Ref)
		return UnknownType{Comment: "ref type"}, nil
	}
	return g.getTypeByName(ft.Type)
}

func (g *Generator) getMemberType(f xmlfile.MemberdefType) (Type, error) {
	if f.Type == nil {
		return UnknownType{}, nil
	} else if len(f.Type.Ref) == 1 {
		ref := f.Type.Ref[0].Refid
		typ := g.types[ref]
		if typ == nil {
			typ = &StructType{BaseNode: BaseNode{refid: ref}} // TODO: UnresolvedType
			g.types[ref] = typ
		}
		return typ, nil
	} else if len(f.Type.Ref) != 0 {
		// FIXME: need to read XML tokens here - refs are mixed with text
		log.Printf("WARN: ref type: %+v", f.Type.Ref)
		return UnknownType{Comment: "ref type"}, nil
	}
	typ := strings.TrimSpace(f.Type.Type + " " + f.Argsstring)
	return g.getTypeByName(typ)
}
