package testutil

import "context"

// PathSet は指定パスの存在チェックを map で模倣する。
type PathSet map[string]bool

func (s PathSet) Exists(_ context.Context, p string) bool { return s[p] }

// CheckerFor は指定パスが存在する PathSet を返す。
func CheckerFor(paths ...string) PathSet {
	s := make(PathSet, len(paths))
	for _, p := range paths {
		s[p] = true
	}
	return s
}

// AlwaysTrue は全パスを存在扱いにする。
type AlwaysTrue struct{}

func (AlwaysTrue) Exists(context.Context, string) bool { return true }

// AlwaysFalse は全パスを不在扱いにする。
type AlwaysFalse struct{}

func (AlwaysFalse) Exists(context.Context, string) bool { return false }
