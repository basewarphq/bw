package shellfiles

import (
	"io/fs"
	"path/filepath"
)

var defaultSkipDirs = map[string]struct{}{
	"node_modules": {},
	".git":         {},
	".svn":         {},
	".hg":          {},
	"vendor":       {},
	".terraform":   {},
	"dist":         {},
	"build":        {},
	".next":        {},
	"__pycache__":  {},
}

type WalkOptions struct {
	SkipDirs   map[string]struct{}
	Extensions []string
}

func DefaultWalkOptions() WalkOptions {
	return WalkOptions{
		SkipDirs: defaultSkipDirs,
	}
}

func WalkFiles(root string, opts WalkOptions, callback func(path string, entry fs.DirEntry) error) error {
	skipDirs := opts.SkipDirs
	if skipDirs == nil {
		skipDirs = defaultSkipDirs
	}

	extSet := make(map[string]struct{}, len(opts.Extensions))
	for _, ext := range opts.Extensions {
		extSet[ext] = struct{}{}
	}

	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if _, skip := skipDirs[entry.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}

		if len(extSet) > 0 {
			ext := filepath.Ext(path)
			if _, ok := extSet[ext]; !ok {
				return nil
			}
		}

		return callback(path, entry)
	})
}

func FindByExtension(root string, extensions ...string) ([]string, error) {
	var files []string
	opts := DefaultWalkOptions()
	opts.Extensions = extensions

	err := WalkFiles(root, opts, func(path string, _ fs.DirEntry) error {
		files = append(files, path)
		return nil
	})

	return files, err
}

func FindShellScripts(root string) ([]string, error) {
	return FindByExtension(root, ".sh")
}
