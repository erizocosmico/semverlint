package semverlint

import (
	"fmt"
	"go/types"
	"strings"
)

type APIChanges []PackageChanges

type PackageChanges struct {
	Name    string
	Path    string
	Changes []Change
}

func NewPackageChanges(name, path string, changes ...Change) PackageChanges {
	return PackageChanges{name, path, changes}
}

type DeclType byte

const (
	VarType DeclType = iota
	ConstType
	FuncType
	InterfaceType
	StructType
	TypeDefType
	PackageType
)

func (d DeclType) String() string {
	switch d {
	case VarType:
		return "package-level variable"
	case ConstType:
		return "package-level constant"
	case FuncType:
		return "function"
	case InterfaceType:
		return "interface"
	case StructType:
		return "struct"
	case TypeDefType:
		return "type definition"
	case PackageType:
		return "package"
	default:
		return "INVALID"
	}
}

type Change interface {
	String() string
}

type DeclChange struct {
	Name    string
	Type    DeclType
	Changes []Change
}

func NewDeclChange(name string, typ DeclType, changes ...Change) DeclChange {
	return DeclChange{name, typ, changes}
}

func (d DeclChange) String() string {
	return fmt.Sprintf(
		"%s %s: %s",
		d.Type,
		d.Name,
		joinChanges(d.Changes),
	)
}

type ArgumentChanged struct {
	Pos     int
	Name    string
	Type    types.Type
	Changes []Change
}

func (a ArgumentChanged) String() string {
	return fmt.Sprintf(
		"argument %s with type %s at position %d: %s",
		a.Name,
		typeString(a.Type),
		a.Pos,
		joinChanges(a.Changes),
	)
}

type ResultChanged struct {
	Pos     int
	Type    types.Type
	Changes []Change
}

func (r ResultChanged) String() string {
	return fmt.Sprintf(
		"result with type %s at position %d: %s",
		typeString(r.Type),
		r.Pos,
		joinChanges(r.Changes),
	)
}

type FieldChanged struct {
	Pos     int
	Name    string
	Changes []Change
}

func (f FieldChanged) String() string {
	return fmt.Sprintf(
		"field %q at position %d: %s",
		f.Name,
		f.Pos,
		joinChanges(f.Changes),
	)
}

type TypeChanged struct {
	From types.Type
	To   types.Type
}

func (tc TypeChanged) String() string {
	return fmt.Sprintf("type changed from %q to %q", tc.From, tc.To)
}

type PositionChanged struct {
	From int
	To   int
}

func (p PositionChanged) String() string {
	return fmt.Sprintf("position changed from %d to %d", p.From, p.To)
}

type Removed struct{}

func (Removed) String() string { return "was removed" }

type Added struct{}

func (Added) String() string { return "was added" }

type ValueChanged struct {
	From string
	To   string
}

func (v ValueChanged) String() string {
	return fmt.Sprintf("value changed from %s to %s", v.From, v.To)
}

func IsBreaking(change Change) bool {
	switch c := change.(type) {
	case Removed,
		PositionChanged,
		TypeChanged,
		FieldChanged,
		ResultChanged,
		ArgumentChanged:
		return true
	case DeclChange:
		for _, c := range c.Changes {
			if IsBreaking(c) {
				return true
			}
		}
	}

	return false
}

func joinChanges(cs []Change) string {
	var strs = make([]string, len(cs))
	for i, c := range cs {
		strs[i] = c.String()
	}
	return strings.Join(strs, ", ")
}

func typeString(t types.Type) string {
	// TODO: check actual printing here is what's required for display
	return t.String()
}
