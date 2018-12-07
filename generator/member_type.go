package generator

import (
	"bytes"
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
		"NSString *":      NSString{},
	}
	primitiveTypes = map[string]string{
		"BOOL":               "bool",
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

func (t PrimitiveType) GoTypeName() (string, bool) {
	return t.Name, true
}

func (t PrimitiveType) CastToObjC(exp string) (string, bool) {
	return exp, true
}

func (t PrimitiveType) CastToGo(exp string) (string, bool) {
	switch t.Name {
	case "bool":
		return exp + ".Bool()", true
	}
	// TODO: more types
	return exp, false
}

type NamedType struct {
	Name string
}

func (t NamedType) GoTypeName() (string, bool) {
	return t.Name, true
}

func (t NamedType) CastToObjC(exp string) (string, bool) {
	return exp, false
}

func (t NamedType) CastToGo(exp string) (string, bool) {
	return exp, false
}

type ConstType struct {
	Type
}

type StrongType struct {
	Type
}

type NullableType struct {
	Type
	Nullable bool
}

// appkitExtern represents an EXTERN macros. It's only used while parsing, it shouldn't appear in the tree.
type appkitExtern struct {
	Type
}

type PtrType struct {
	Elem Type
}

func (t PtrType) GoTypeName() (string, bool) {
	e, ok := t.Elem.GoTypeName()
	return "*" + e, ok
}

func (t PtrType) CastToObjC(exp string) (string, bool) {
	return t.Elem.CastToObjC(exp)
}

func (t PtrType) CastToGo(exp string) (string, bool) {
	return t.Elem.CastToGo(exp)
}

type ArrayType struct {
	Size string // TODO: expression
	Elem Type
}

func (t ArrayType) GoTypeName() (string, bool) {
	e, ok := t.Elem.GoTypeName()
	return "[" + t.Size + "]" + e, ok
}

func (t ArrayType) CastToObjC(exp string) (string, bool) {
	return exp, false
}

func (t ArrayType) CastToGo(exp string) (string, bool) {
	return exp, false
}

type FuncType struct {
	Return Type
	Args   []*FuncArg
}

func (t *FuncType) GoTypeName() (string, bool) {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("func")
	ok := t.printGoArgs(buf)
	return buf.String(), ok
}

func (t *FuncType) CastToObjC(exp string) (string, bool) {
	return exp, false // TODO(dennwc): cannot convert functions, need to write cgo wrappers
}

func (t *FuncType) CastToGo(exp string) (string, bool) {
	return exp, false
}

type FuncArg struct {
	Name string
	Type Type
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
		if a.Type == nil {
			ok = false
			continue
		}
		at, ok2 := a.Type.GoTypeName()
		if !ok2 {
			ok = false
		}
		fmt.Fprint(w, at)
	}
	fmt.Fprint(w, ")")
	if t.Return != nil {
		rt, ok2 := t.Return.GoTypeName()
		if !ok2 {
			ok = false
		}
		fmt.Fprint(w, " "+rt)
	}
	return ok
}

type UnknownType struct {
	Name    string
	Comment string
}

func (t UnknownType) GoTypeName() (string, bool) {
	typ := "interface{}"
	if t.Name != "" {
		typ = t.Name
	}
	if t.Comment != "" {
		typ += " /* " + t.Comment + " */"
	}
	return typ, false
}

func (t UnknownType) CastToObjC(exp string) (string, bool) {
	return exp, false
}

func (t UnknownType) CastToGo(exp string) (string, bool) {
	return exp, false
}

var prefixWrappers = []struct {
	prefix string
	suffix string
	wrap   func(Type) Type
}{
	{prefix: "APPKIT_EXTERN ", wrap: func(e Type) Type {
		return appkitExtern{e}
	}},
	{prefix: "__nullable ", wrap: func(e Type) Type {
		return NullableType{Type: e, Nullable: true}
	}},
	{prefix: "nullable ", wrap: func(e Type) Type {
		return NullableType{Type: e, Nullable: true}
	}},
	{prefix: "__null_unspecified ", wrap: func(e Type) Type {
		return e // TODO
	}},
	{prefix: "const ", wrap: func(e Type) Type {
		return ConstType{e}
	}},
	{prefix: "__strong ", wrap: func(e Type) Type {
		return StrongType{e}
	}},
	{suffix: "*", wrap: func(e Type) Type {
		return PtrType{Elem: e}
	}},
	{suffix: "_Nullable", wrap: func(e Type) Type {
		return NullableType{Type: e, Nullable: true}
	}},
	{suffix: "__nonnull", wrap: func(e Type) Type {
		return NullableType{Type: e, Nullable: false}
	}},
	{suffix: "__nullable", wrap: func(e Type) Type {
		return NullableType{Type: e, Nullable: true}
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
