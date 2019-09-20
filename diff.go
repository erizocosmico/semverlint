package semverlint

import "go/types"

// Diff computes the difference between two given public APIs.
func Diff(current, prev API) APIChanges {
	var changes APIChanges
	currentPkgs := packagesIndex(current)
	prevPkgs := packagesIndex(prev)

	var seen = make(map[string]struct{})
	for path, p1 := range prevPkgs {
		seen[path] = struct{}{}
		p2, ok := currentPkgs[path]
		if !ok {
			changes = append(changes, NewPackageChanges(
				p1.Name, p1.Path,
				NewDeclChange(p1.Name, PackageType, Removed{}),
			))
			continue
		}

		changes = append(changes, packageDiff(p2, p1))
	}

	// Add the packages that were not present as new.
	for path, p := range currentPkgs {
		// Skip packages we've already seen.
		if _, ok := seen[path]; !ok {
			changes = append(changes, NewPackageChanges(
				p.Name, p.Path,
				NewDeclChange(p.Name, PackageType, Added{}),
			))
		}
	}

	return changes
}

func packageDiff(prev, current Package) PackageChanges {
	var changes []Change
	changes = append(changes, constsDiff(prev.Consts, current.Consts)...)
	changes = append(changes, varsDiff(prev.Vars, current.Vars)...)
	changes = append(changes, funcsDiff(prev.Funcs, current.Funcs)...)
	changes = append(changes, structsDiff(prev.Structs, current.Structs)...)
	changes = append(changes, interfacesDiff(prev.Interfaces, current.Interfaces)...)
	changes = append(changes, typesDiff(prev.Types, current.Types)...)
	return PackageChanges{
		Path:    current.Path,
		Name:    current.Name,
		Changes: changes,
	}
}

func constsDiff(prev, current []Const) []Change {
	var changes []Change
	currentConsts := constsIndex(current)
	prevConsts := constsIndex(prev)

	var seen = make(map[string]struct{})
	for name, v := range prevConsts {
		seen[name] = struct{}{}
		v2, ok := currentConsts[name]
		if !ok {
			changes = append(changes, NewDeclChange(name, ConstType, Removed{}))
		}

		if !typesEqual(v.Type, v2.Type) {
			changes = append(changes, NewDeclChange(name, ConstType, TypeChanged{
				From: v.Type,
				To:   v2.Type,
			}))
		}

		if v.Value != v2.Value {
			changes = append(changes, NewDeclChange(name, ConstType, ValueChanged{
				From: v.Value,
				To:   v2.Value,
			}))
		}
	}

	for name := range currentConsts {
		if _, ok := seen[name]; !ok {
			changes = append(changes, NewDeclChange(name, ConstType, Added{}))
		}
	}

	return changes
}

func varsDiff(prev, current []Var) []Change {
	var changes []Change
	currentVars := varsIndex(current)
	prevVars := varsIndex(prev)

	var seen = make(map[string]struct{})
	for name, v := range prevVars {
		seen[name] = struct{}{}
		v2, ok := currentVars[name]
		if !ok {
			changes = append(changes, NewDeclChange(name, VarType, Removed{}))
		} else if !typesEqual(v.Type, v2.Type) {
			changes = append(changes, NewDeclChange(name, VarType, TypeChanged{
				From: v.Type,
				To:   v2.Type,
			}))
		}
	}

	for name := range currentVars {
		if _, ok := seen[name]; !ok {
			changes = append(changes, NewDeclChange(name, VarType, Added{}))
		}
	}

	return changes
}

func funcsDiff(prev, current []Func) []Change {
	var changes []Change
	currentFuncs := funcsIndex(current)
	prevFuncs := funcsIndex(prev)

	var seen = make(map[string]struct{})
	for name, v := range prevFuncs {
		seen[name] = struct{}{}
		v2, ok := currentFuncs[name]
		if !ok {
			changes = append(changes, NewDeclChange(name, FuncType, Removed{}))
		}

		_ = v
		_ = v2

		// TODO: check args

		// TODO: check returns
	}

	for name := range currentFuncs {
		if _, ok := seen[name]; !ok {
			changes = append(changes, NewDeclChange(name, FuncType, Added{}))
		}
	}

	return changes
}

func structsDiff(prev, current []Struct) []Change {
	var changes []Change
	currentStructs := structsIndex(current)
	prevStructs := structsIndex(prev)

	var seen = make(map[string]struct{})
	for name, v := range prevStructs {
		seen[name] = struct{}{}
		v2, ok := currentStructs[name]
		if !ok {
			changes = append(changes, NewDeclChange(name, StructType, Removed{}))
		}

		_ = v
		_ = v2

		// TODO: check fields

		// TODO: check methods
	}

	for name := range currentStructs {
		if _, ok := seen[name]; !ok {
			changes = append(changes, NewDeclChange(name, StructType, Added{}))
		}
	}

	return changes
}

func interfacesDiff(prev, current []Interface) []Change {
	var changes []Change
	currentInterfaces := interfacesIndex(current)
	prevInterfaces := interfacesIndex(prev)

	var seen = make(map[string]struct{})
	for name, v := range prevInterfaces {
		seen[name] = struct{}{}
		v2, ok := currentInterfaces[name]
		if !ok {
			changes = append(changes, NewDeclChange(name, InterfaceType, Removed{}))
		}

		_ = v
		_ = v2

		// TODO: check methods
	}

	for name := range currentInterfaces {
		if _, ok := seen[name]; !ok {
			changes = append(changes, NewDeclChange(name, InterfaceType, Added{}))
		}
	}

	return changes
}

func typesDiff(prev, current []TypeDef) []Change {
	var changes []Change
	currentTypes := typesIndex(current)
	prevTypes := typesIndex(prev)

	var seen = make(map[string]struct{})
	for name, v := range prevTypes {
		seen[name] = struct{}{}
		v2, ok := currentTypes[name]
		if !ok {
			changes = append(changes, NewDeclChange(name, TypeDefType, Removed{}))
		}

		if !typesEqual(v.Type, v2.Type) {
			changes = append(changes, NewDeclChange(name, TypeDefType, TypeChanged{
				From: v.Type,
				To:   v2.Type,
			}))
		}
	}

	for name := range currentTypes {
		if _, ok := seen[name]; !ok {
			changes = append(changes, NewDeclChange(name, TypeDefType, Added{}))
		}
	}

	return changes
}

func packagesIndex(a API) map[string]Package {
	var result = make(map[string]Package)
	for _, p := range a {
		result[p.Path] = p
	}
	return result
}

func constsIndex(xs []Const) map[string]Const {
	var result = make(map[string]Const)
	for _, x := range xs {
		result[x.Name] = x
	}
	return result
}

func varsIndex(xs []Var) map[string]Var {
	var result = make(map[string]Var)
	for _, x := range xs {
		result[x.Name] = x
	}
	return result
}

func funcsIndex(xs []Func) map[string]Func {
	var result = make(map[string]Func)
	for _, x := range xs {
		result[x.Name] = x
	}
	return result
}

func interfacesIndex(xs []Interface) map[string]Interface {
	var result = make(map[string]Interface)
	for _, x := range xs {
		result[x.Name] = x
	}
	return result
}

func structsIndex(xs []Struct) map[string]Struct {
	var result = make(map[string]Struct)
	for _, x := range xs {
		result[x.Name] = x
	}
	return result
}

func typesIndex(xs []TypeDef) map[string]TypeDef {
	var result = make(map[string]TypeDef)
	for _, x := range xs {
		result[x.Name] = x
	}
	return result
}

func typesEqual(a, b types.Type) bool {
	return false
}
