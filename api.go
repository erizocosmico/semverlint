package semverlint

import (
	"fmt"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"golang.org/x/tools/go/packages"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// Version of a project.
type Version struct {
	Name   string
	Commit plumbing.Hash
}

// Versions returns a list of versions for the repository at the given path.
func Versions(path string) ([]Version, error) {
	var result []Version

	r, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open repository: %s", err)
	}

	head, err := r.Head()
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return nil, fmt.Errorf("no HEAD reference found in repository")
		}

		return nil, fmt.Errorf("unable to get HEAD of repository: %s", err)
	}

	result = append(result, Version{"HEAD", head.Hash()})

	iter, err := r.Tags()
	if err != nil {
		return nil, fmt.Errorf("unable to list tags of repository: %s", err)
	}

	for {
		tag, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, fmt.Errorf("error getting next tag: %s", err)
		}

		if _, err := r.CommitObject(tag.Hash()); err != nil {
			// skip tags not pointing to commits
			if err == plumbing.ErrObjectNotFound {
				continue
			}

			return nil, fmt.Errorf("unknown error getting commit: %s", err)
		}

		// skip tags which are not valid semver versions
		if _, err := semver.NewVersion(tag.Name().Short()); err != nil {
			continue
		}

		result = append(result, Version{tag.Name().Short(), tag.Hash()})
	}

	sort.Stable(byVersion(result))
	return result, nil
}

type byVersion []Version

func (b byVersion) Len() int      { return len(b) }
func (b byVersion) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byVersion) Less(i, j int) bool {
	if b[i].Name == "HEAD" {
		return true
	}

	if b[j].Name == "HEAD" {
		return false
	}

	// We know for a fact that they're correct semver versions, since this was
	// already checked when adding to the list of versions.
	v1 := semver.MustParse(b[i].Name)
	v2 := semver.MustParse(b[j].Name)

	return v1.LessThan(v2)
}

// API that is exposed on a project.
type API []Package

// VersionAPI returns the public API of the project at the given path at the
// given version.
func VersionAPI(path string, version Version) (API, error) {
	return nil, fmt.Errorf("not implemented")
}

// ProjectAPI returns the public API of the project at the given path.
func ProjectAPI(path string) (API, error) {
	packages, err := projectPackages(path)
	if err != nil {
		return nil, fmt.Errorf("error getting project packages: %s", err)
	}

	var api API
	for _, pkg := range packages {
		p, err := packageFromGoPackage(pkg)
		if err != nil {
			return nil, fmt.Errorf("error converting from Go package to internal package: %s", err)
		}
		api = append(api, p)
	}

	return api, nil
}

func projectDirs(path string) ([]string, error) {
	var dirs = make(map[string]struct{})
	err := filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return filepath.SkipDir
		}

		// Skip vendor and examples directory.
		if fi.IsDir() && (fi.Name() == "vendor" || fi.Name() == "_examples") {
			return filepath.SkipDir
		}

		if !fi.IsDir() {
			dir, file := filepath.Dir(p), filepath.Base(p)
			// Exclude tests and non-Go files.
			if strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, "_test.go") {
				dirs[dir] = struct{}{}
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to walk project tree: %s", err)
	}

	var dirNames = make([]string, 0, len(dirs))
	for d := range dirs {
		dirNames = append(dirNames, d)
	}

	sort.Strings(dirNames)
	return dirNames, nil
}

func projectPackages(path string) ([]*types.Package, error) {
	dirs, err := projectDirs(path)
	if err != nil {
		return nil, err
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode:  packages.NeedTypes,
		Tests: false,
	}, dirs...)
	if err != nil {
		return nil, fmt.Errorf("can't load packages: %s", err)
	}

	var result = make([]*types.Package, len(pkgs))
	for i, p := range pkgs {
		result[i] = p.Types
	}

	return result, nil
}

func packageFromGoPackage(gopkg *types.Package) (Package, error) {
	name, path, scope := gopkg.Name(), gopkg.Path(), gopkg.Scope()
	pkg := Package{Name: name, Path: path}
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}

		switch obj := obj.(type) {
		case *types.Func:
			pkg.Funcs = append(pkg.Funcs, funcFromGoFunc(obj))
		case *types.TypeName:
			switch t := obj.Type().(type) {
			case *types.Interface:
				iface := Interface{Name: obj.Name()}
				for i := 0; i < t.NumMethods(); i++ {
					method := funcFromGoFunc(t.Method(i))
					iface.Methods = append(iface.Methods, method)
				}
				pkg.Interfaces = append(pkg.Interfaces, iface)
			case *types.Struct:
				s := Struct{Name: obj.Name()}
				for i := 0; i < t.NumFields(); i++ {
					f := t.Field(i)
					s.Fields = append(s.Fields, Field{
						Name: f.Name(),
						Type: f.Type(),
					})
				}
				for _, t := range []types.Type{obj.Type(), types.NewPointer(obj.Type())} {
					mset := types.NewMethodSet(t)
					for i := 0; i < mset.Len(); i++ {
						method := funcFromGoFunc(mset.At(i).Obj().(*types.Func))
						s.Methods = append(s.Methods, method)
					}
				}
				pkg.Structs = append(pkg.Structs, s)
			default:
				pkg.Types = append(pkg.Types, TypeDef{
					Name:  obj.Name(),
					Type:  t,
					Alias: obj.IsAlias(),
				})
			}
		case *types.Var:
			pkg.Vars = append(pkg.Vars, Var{
				Name: obj.Name(),
				Type: obj.Type(),
			})
		case *types.Const:
			pkg.Consts = append(pkg.Consts, Const{
				Name:  obj.Name(),
				Type:  obj.Type(),
				Value: obj.Val().ExactString(),
			})
		}
	}

	return pkg, nil
}

func funcFromGoFunc(obj *types.Func) Func {
	sig := obj.Type().(*types.Signature)
	var args = make([]types.Type, sig.Params().Len())
	for i := 0; i < sig.Params().Len(); i++ {
		args[i] = sig.Params().At(i).Type()
	}

	var results = make([]types.Type, sig.Results().Len())
	for i := 0; i < sig.Results().Len(); i++ {
		results[i] = sig.Results().At(i).Type()
	}

	return Func{
		Name:   obj.Name(),
		Args:   args,
		Return: results,
	}
}
