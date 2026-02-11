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

// AllPathsExist は全パスを存在扱いにする PathChecker スタブ。
type AllPathsExist struct{}

func (AllPathsExist) Exists(context.Context, string) bool { return true }

// NoPathsExist は全パスを不在扱いにする PathChecker スタブ。
type NoPathsExist struct{}

func (NoPathsExist) Exists(context.Context, string) bool { return false }
