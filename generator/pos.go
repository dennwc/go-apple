package generator

import (
	"fmt"

	"github.com/dennwc/go-doxy/xmlfile"
)

type Location struct {
	File string
	Line int
	Col  int
}

func (l *Location) String() string {
	if l == nil {
		return ""
	}
	if l.Col <= 1 {
		return fmt.Sprintf("%s:%d", l.File, l.Line)
	}
	return fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Col)
}

type LineRange struct {
	File      string
	StartLine int
	EndLine   int
}

func asLocation(l xmlfile.LocationType) *Location {
	if l.File == "" {
		return nil
	}
	return &Location{
		File: l.File,
		Line: l.Line,
		Col:  l.Column,
	}
}

func asLineRange(l xmlfile.LocationType) *LineRange {
	if l.Bodyfile == "" || l.Bodyend < 0 {
		return nil
	}
	return &LineRange{
		File:      l.Bodyfile,
		StartLine: l.Bodystart,
		EndLine:   l.Bodyend,
	}
}
