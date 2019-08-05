package ace

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
)

// Special characters
const (
	cr   = "\r"
	lf   = "\n"
	crlf = "\r\n"

	space        = " "
	equal        = "="
	pipe         = "|"
	doublePipe   = pipe + pipe
	slash        = "/"
	sharp        = "#"
	dot          = "."
	doubleDot    = dot + dot
	colon        = ":"
	doubleColon  = colon + colon
	doubleQuote  = `"`
	lt           = "<"
	gt           = ">"
	exclamation  = "!"
	hyphen       = "-"
	bracketOpen  = "["
	bracketClose = "]"
)

var (
	sourceCache      = make(map[string]*source)
	sourceCacheMutex = new(sync.RWMutex)
)

// readFiles reads files and returns source for the parsing process.
func readFiles(basePath, innerPath string, opts *Options) (*source, error) {
	name := basePath + colon + innerPath
	if !opts.SourceCache {
		if src, ok := getSourceCache(name); ok {
			return src, nil
		}
	}

	// Read the base file.
	base, err := readFile(basePath, opts)
	if err != nil {
		return nil, err
	}

	// Read the inner file.
	inner, err := readFile(innerPath, opts)
	if err != nil {
		return nil, err
	}

	var includes []*File

	// Find include files from the base file.
	if err := findIncludes(base.data, opts, &includes, base); err != nil {
		return nil, err
	}

	// Find include files from the inner file.
	if err := findIncludes(inner.data, opts, &includes, inner); err != nil {
		return nil, err
	}

	src := NewSource(base, inner, includes)

	if !opts.SourceCache {
		setSourceCache(name, src)
	}

	return src, nil
}

// readFile reads a file and returns a file struct.
func readFile(path string, opts *Options) (*File, error) {
	var data []byte
	var err error

	if len(path) > 0 {
		ds := append([]string{}, opts.BaseDirs...)
		if len(ds) == 0 {
			ds = append(ds, "")
		}
		for _, v := range ds {
			name := filepath.Join(v, path+dot+opts.Extension)
			if opts.Asset != nil {
				if data, err = opts.Asset(name); err == nil {
					return NewFile(path, data), nil
				}
				continue
			}
			if data, err = ioutil.ReadFile(name); err == nil {
				return NewFile(path, data), nil
			}
		}

		return nil, err
	}

	return NewFile(path, data), nil
}

// findIncludes finbs and adds include files.
func findIncludes(data []byte, opts *Options, includes *[]*File, targetFile *File) error {
	includePaths, err := findIncludePaths(data, opts, targetFile)
	if err != nil {
		return err
	}

	for _, includePath := range includePaths {
		if !hasFile(*includes, includePath) {
			f, err := readFile(includePath, opts)
			if err != nil {
				return err
			}

			*includes = append(*includes, f)

			if err := findIncludes(f.data, opts, includes, f); err != nil {
				return err
			}
		}
	}

	return nil
}

// findIncludePaths finds and returns include paths.
func findIncludePaths(data []byte, opts *Options, f *File) ([]string, error) {
	var includePaths []string

	for i, str := range strings.Split(formatLF(string(data)), lf) {
		ln := newLine(i+1, str, opts, f)

		if ln.isHelperMethodOf(helperMethodNameInclude) {
			if len(ln.tokens) < 3 {
				return nil, fmt.Errorf("no template name is specified [file: %s][line: %d]", ln.fileName(), ln.no)
			}

			includePaths = append(includePaths, ln.tokens[2])
		}
	}

	return includePaths, nil
}

// formatLF replaces the line feed codes with LF and returns the result.
func formatLF(s string) string {
	return strings.Replace(strings.Replace(s, crlf, lf, -1), cr, lf, -1)
}

// hasFile return if files has a file which has the path specified by the parameter.
func hasFile(files []*File, path string) bool {
	for _, f := range files {
		if f.path == path {
			return true
		}
	}

	return false
}

func getSourceCache(name string) (*source, bool) {
	sourceCacheMutex.RLock()
	src, ok := sourceCache[name]
	sourceCacheMutex.RUnlock()
	return src, ok
}

func setSourceCache(name string, src *source) {
	sourceCacheMutex.Lock()
	sourceCache[name] = src
	sourceCacheMutex.Unlock()
}

func FlushSourceCache() {
	sourceCacheMutex.Lock()
	sourceCache = make(map[string]*source)
	sourceCacheMutex.Unlock()
}
