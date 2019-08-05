package ace

import (
	"html/template"
	"sync"
)

var (
	templateCache      = make(map[string]template.Template)
	templateCacheMutex = new(sync.RWMutex)
)

// Load loads and returns an HTML template. Each Ace templates are parsed only once
// and cached if the "DynamicReload" option are not set.
func Load(basePath, innerPath string, opts *Options) (*template.Template, error) {
	// Initialize the options.
	opts = InitializeOptions(opts)
	name := basePath + colon + innerPath

	if opts.TemplateCache {
		if tpl, ok := getTemplateCache(name); ok {
			return &tpl, nil
		}
	}

	// Read files.
	src, err := readFiles(basePath, innerPath, opts)
	if err != nil {
		return nil, err
	}

	// Parse the source.
	rslt, err := ParseSource(src, opts)
	if err != nil {
		return nil, err
	}

	// Compile the parsed result.
	tpl, err := CompileResult(name, rslt, opts)
	if err != nil {
		return nil, err
	}

	if opts.TemplateCache {
		setTemplateCache(name, *tpl)
	}

	return tpl, nil
}

// getTemplateCache returns the cached template.
func getTemplateCache(name string) (template.Template, bool) {
	templateCacheMutex.RLock()
	tpl, ok := templateCache[name]
	templateCacheMutex.RUnlock()
	return tpl, ok
}

// setTemplateCache sets the template to the templateCache.
func setTemplateCache(name string, tpl template.Template) {
	templateCacheMutex.Lock()
	templateCache[name] = tpl
	templateCacheMutex.Unlock()
}

// FlushTemplateCache clears all cached templates.
func FlushTemplateCache() {
	templateCacheMutex.Lock()
	templateCache = make(map[string]template.Template)
	templateCacheMutex.Unlock()
}
