package treeloader

import (
	"errors"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	multierror "github.com/hashicorp/go-multierror"

	fsnotify "gopkg.in/fsnotify.v1"
)

var (
	// DefaultExtensions limits the watch to only go files
	DefaultExtensions = StringSet{".go": 1}

	// ErrMustIncludeCommand indicates that no command was passed in
	ErrMustIncludeCommand = errors.New("must include a command")
	// ErrTooDeep indicates that the maxDepth has been reached
	ErrTooDeep = errors.New("too deep")

	sleepDuration = 20 * time.Millisecond
	maxDepth      = 100
	baseDir       = path.Join(build.Default.GOPATH, "src")
)

// NewRunner creates a runner for the command provided
func NewRunner(t *Treeloader) *Runner {
	return &Runner{
		t:          t,
		cmdPath:    t.options.CmdPath,
		executable: strings.Replace(t.options.CmdPath, ".go", "", 1),
		logger:     t.logger,
	}
}

// Runner manages building, running, and killing commands
type Runner struct {
	cmdPath    string
	executable string
	cmd        *exec.Cmd
	logger     loggerFunc
	t          *Treeloader
}

// Run builds and runs the go command provided
func (r *Runner) Run() *Runner {
	if !r.t.lastRun.IsZero() {
		timeSince := time.Now().Sub(r.t.lastRun)
		detailedLog("time since last build: %s", timeSince)
	}
	r.t.lastRun = time.Now()
	r.logger("building: %s", r.cmdPath)
	buildCmd := exec.Command("go", "build", "-o", r.executable, r.cmdPath)
	buildCmd.Stderr = os.Stderr
	buildCmd.Stdout = os.Stdout
	if err := buildCmd.Run(); err != nil {
		detailedLog("error building: %s", err)
		r.t.previousError = true
		return r
	}

	r.logger("running: %s", r.executable)
	r.cmd = exec.Command(r.executable)
	r.cmd.Stderr = os.Stderr
	r.cmd.Stdout = os.Stdout
	if err := r.cmd.Start(); err != nil {
		r.t.previousError = true
		detailedLog("error running: %s", err)
		return r
	}
	if r.t.previousError {
		r.t.previousError = false
		detailedLog("successful build and run")
	}
	return r
}

// Running indicates whether the watched program is already running
func (r *Runner) Running() bool {
	if r.cmd == nil {
		return false
	}
	return r.cmd.ProcessState != nil && !r.cmd.ProcessState.Exited()
}

// Kill blocks until the running command is killed.
func (r *Runner) Kill() error {
	r.logger("killing %d", r.cmd.Process.Pid)
	return r.cmd.Process.Kill()
}

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
		return nil, err
	}

	if !pkg.IsCommand() {
		return nil, errors.New("package must be a command")
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
			return err
		}
		if importedPkg.Goroot {
			continue
		}
		dirs.Add(imp)

		if err = addImports(dirs, importedPkg, depth+1); err != nil {
			return err
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
	// ensure each extension is prefixed with a period
	if len(wo.Extensions) > 0 {
		for ext := range wo.Extensions {
			if !strings.HasPrefix(ext, ".") {
				wo.Extensions.Add("." + ext)
				wo.Extensions.Remove(ext)
			}
		}
	} else {
		wo.Extensions = DefaultExtensions
	}
	return nil
}

type Treeloader struct {
	options       *Options
	watcher       *fsnotify.Watcher
	runner        *Runner
	logger        loggerFunc
	lastRun       time.Time
	previousError bool
}

func New(opts *Options) (*Treeloader, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	return &Treeloader{
		options: opts,
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

func (t *Treeloader) ExitOnErr(err error) {
	if err != nil {
		if closeErr := t.Close(); closeErr != nil {
			detailedLog("error closing: %s", closeErr)
		}
		detailedLog("fatal error: %s", err)
	}
}

// Run calculates directories to watch and runs the command every time a file is changed
func (t *Treeloader) Run() error {
	t.logger = makeLogger(t.options.Verbose)
	depChan := make(chan string, 1)

	go func() {
		for {
			select {
			case f := <-depChan:
				if t.runner != nil && t.runner.Running() {
					t.runner.Kill()
				}
				t.runner = NewRunner(t).Run()

				dirs, err := DirsToWatch(t.options.CmdPath)
				t.ExitOnErr(err)

				if t.options.Verbose {
					t.logger(dirs.String())
				}
				t.watchFiles(dirs, depChan, t.logger)

				if t.options.Reloaded != nil {
					select {
					case t.options.Reloaded <- f:
					default:
						fmt.Println("Channel full. Discarding value")
					}
				}
			}
		}
	}()

	depChan <- ""

	for {
		time.Sleep(sleepDuration)
	}
}

func (t *Treeloader) watchFiles(deps map[string]int, depChan chan<- string, logger loggerFunc) {
	if t.watcher != nil {
		t.ExitOnErr(t.watcher.Close())
	}
	var err error
	t.watcher, err = fsnotify.NewWatcher()
	t.ExitOnErr(err)

	go func() {
		for {
			select {
			case event := <-t.watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write && t.options.Extensions.Contains(filepath.Ext(event.Name)) {
					depChan <- event.Name

				}
			case err := <-t.watcher.Errors:
				t.ExitOnErr(err)
			}
		}
	}()

	for d := range deps {
		logger("watching %s", d)
		t.ExitOnErr(t.watcher.Add(path.Join(baseDir, d)))
	}
}
