package treeloader

import (
	"errors"
	"go/build"
	"path"
	"path/filepath"
	"strings"
	"time"

	multierror "github.com/hashicorp/go-multierror"

	"github.com/richardwilkes/errs"
	fsnotify "gopkg.in/fsnotify.v1"
)

var (
	// ErrMustIncludeCommand indicates that no command was passed in
	ErrMustIncludeCommand = errors.New("must include a command")
	// ErrTooDeep indicates that the maxDepth has been reached
	ErrTooDeep = errors.New("too deep")

	sleepDuration = 20 * time.Millisecond
	maxDepth      = 100
	baseDir       = path.Join(build.Default.GOPATH, "src")
)

// DirsToWatch traverses the dependency tree to determine what files to watch
func DirsToWatch(entry string) (StringSet, error) {
	entry, err := filepath.Abs(entry)
	if err != nil {
		return nil, err
	}

	entry, err = filepath.Rel(baseDir, entry)
	if err != nil {
		return nil, err
	}

	pkg, err := build.Import(filepath.Dir(entry), ".", 0)
	if err != nil {
		return nil, errs.NewWithCause("eror parsing main package", err)
	}

	if !pkg.IsCommand() {
		return nil, errs.New("package must be a command")
	}

	dirs := StringSet{}
	dirs.Add(filepath.Dir(entry))
	err = addImports(dirs, pkg, 0)

	return dirs, err
}

func addImports(dirs StringSet, pkg *build.Package, depth int) error {
	if depth > maxDepth {
		return ErrTooDeep
	}

	if _, ok := dirs[pkg.Name]; ok {
		return nil
	}
	for _, imp := range pkg.Imports {
		importedPkg, err := build.Import(imp, ".", 0)
		if err != nil {
			return errs.NewWithCause("error parsing package", err)
		}
		if importedPkg.Goroot {
			continue
		}
		dirs.Add(imp)

		if err = addImports(dirs, importedPkg, depth+1); err != nil {
			return errs.NewWithCause("error adding package imports", err)
		}
	}

	return nil
}

// Options are the options available for use when watching files
type Options struct {
	CmdPath    string
	Verbose    bool
	Extensions StringSet
	Reloaded   chan string
}

func (wo *Options) validate() error {
	if wo.CmdPath == "" {
		return ErrMustIncludeCommand
	}

	if len(wo.Extensions) == 0 {
		wo.Extensions = make(StringSet)
	}
	wo.Extensions.Add("go")
	return nil
}

type watchChange struct {
	add  bool
	path string
}

type Treeloader struct {
	options       *Options
	watcher       *fsnotify.Watcher
	runner        *GoCmd
	logger        loggerFunc
	lastRun       time.Time
	previousError bool

	errChan        chan error
	watchChange    chan *watchChange
	currentWatches StringSet
}

func New(opts *Options) (*Treeloader, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Treeloader{
		options:     opts,
		watcher:     watcher,
		errChan:     make(chan error, 1),
		watchChange: make(chan *watchChange, 1),
	}, nil
}

func (t *Treeloader) Close() error {
	var err error
	if t.runner != nil && t.runner.Running() {
		if rErr := t.runner.Kill(); rErr != nil {
			err = multierror.Append(err, rErr)
		}
	}
	if t.watcher != nil {
		if tErr := t.watcher.Close(); tErr != nil {
			err = multierror.Append(err, tErr)
		}
	}
	return err
}

func (t *Treeloader) shouldReload() bool {
	return true
}

// Run calculates directories to watch and runs the command every time a file is changed
func (t *Treeloader) Run() error {
	t.logger = makeLogger(t.options.Verbose)
	// changedFile := make(chan string, 1)

	go func() {
		for {
			select {
			case err := <-t.watcher.Errors:
				if err != nil {
					t.logger("watcher error: %s", err)
				}
			case err := <-t.errChan:
				if err != nil {
					t.logger("error: %s", err)
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case wc := <-t.watchChange:
				op := t.watcher.Add
				msg := "watching: %s"
				if !wc.add {
					op = t.watcher.Remove
					msg = "un" + msg
				}
				t.logger(msg, wc.path)
				t.errChan <- op(wc.path)
			}
		}
	}()

	go t.runNew()
	t.updateWatch()

	// go func() {
	for {
		select {
		case event := <-t.watcher.Events:
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}

			f := event.Name
			if !t.options.Extensions.Contains(strings.Replace(filepath.Ext(f), ".", "", 1)) {
				continue
			}

			if !t.shouldReload() {
				continue
			}

			t.logger("%s changed, restarting", f)
			t.runNew()
			t.updateWatch()
			select {
			case t.options.Reloaded <- f:
			default:
			}
		}
	}
	// }()

	// for {
	// 	time.Sleep(sleepDuration)
	// }
}

func (t *Treeloader) runNew() {
	if t.runner != nil { //&& t.runner.Running()
		if killErr := t.runner.Kill(); killErr != nil {
			t.errChan <- killErr
		}
	}

	t.runner = NewGoCmd(t)
	t.errChan <- t.runner.Run()
}

func (t *Treeloader) updateWatch() {
	newDirs, err := DirsToWatch(t.options.CmdPath)
	if err != nil {
		t.errChan <- err
		return
	}

	if len(t.currentWatches) > 0 {
		removeDirs := t.currentWatches.Difference(newDirs)
		for _, d := range removeDirs {
			t.watchChange <- &watchChange{
				path: d,
			}
		}
	}

	for d := range newDirs {
		// dpath := path.Join(baseDir, d)
		t.watchChange <- &watchChange{
			add:  true,
			path: d,
		}
	}
}
