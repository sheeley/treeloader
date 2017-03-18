package treeloader

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/richardwilkes/errs"
)

// NewGoCmd creates a runner for the command provided
func NewGoCmd(t *Treeloader) *GoCmd {
	return &GoCmd{
		t:          t,
		cmdPath:    t.options.CmdPath,
		executable: strings.Replace(t.options.CmdPath, ".go", "", 1),
		logger:     t.logger,
		errChan:    t.errChan,
	}
}

// GoCmd manages building, running, and killing commands
type GoCmd struct {
	cmdPath    string
	executable string
	cmd        *exec.Cmd
	running    bool
	logger     loggerFunc
	t          *Treeloader
	errChan    chan error
}

// Run builds and runs the go command provided
func (r *GoCmd) Run() error {
	if !r.t.lastRun.IsZero() {
		timeSince := time.Now().Sub(r.t.lastRun)
		logf("time since last build: %s", timeSince)
	}
	r.t.lastRun = time.Now()
	r.logger("running: %s", r.cmdPath)
	// if err := os.Chdir(path.Dir(r.cmdPath)); err != nil {
	// 	detailedLog("error chdir-ing %s", err)
	// }
	r.cmd = exec.Command("go", "run", r.cmdPath)
	r.cmd.SysProcAttr = &syscall.SysProcAttr{
		// Setpgid: true,
		Setsid: true,
	}
	r.cmd.Stderr = os.Stderr
	r.cmd.Stdout = os.Stdout
	return r.cmd.Start()
	// ; err != nil {
	// 	logf("error running: %s", err)
	// 	r.t.previousError = true
	// 	return r
	// }

	// r.running = true
	// if r.t.previousError {
	// 	r.t.previousError = false
	// 	logf("successful build and run")
	// }
	// return r
}

// Run builds and runs the go command provided
// func (r *GoCmd) Run() *GoCmd {
// 	if !r.t.lastRun.IsZero() {
// 		timeSince := time.Now().Sub(r.t.lastRun)
// 		detailedLog("time since last build: %s", timeSince)
// 	}
// 	r.t.lastRun = time.Now()
// 	r.logger("building: %s", r.cmdPath)
// 	buildCmd := exec.Command("go", "build", "-o", r.executable, r.cmdPath)

// 	buildCmd.Stderr = os.Stderr
// 	buildCmd.Stdout = os.Stdout
// 	if err := buildCmd.Run(); err != nil {
// 		detailedLog("error building: %s", err)
// 		r.t.previousError = true
// 		return r
// 	}

// 	r.logger("running: %s", r.executable)
// 	r.cmd = exec.Command(r.executable)
// 	r.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
// 	r.cmd.Stderr = os.Stderr
// 	r.cmd.Stdout = os.Stdout
// 	if err := r.cmd.Start(); err != nil {
// 		r.t.previousError = true
// 		detailedLog("error running: %s", err)
// 		return r
// 	}
// 	r.running = true
// 	if r.t.previousError {
// 		r.t.previousError = false
// 		detailedLog("successful build and run")
// 	}
// 	return r
// }

// Running indicates whether the watched program is already running
func (r *GoCmd) Running() bool {
	return r.running
}

// Kill blocks until the running command is killed.
func (r *GoCmd) Kill() error {
	if !r.running {
		return nil
	}
	// log.Printf("killing %d\n", r.cmd.Process.Pid)
	// pgid, err := syscall.Getpgid(r.cmd.Process.Pid)
	// if err != nil {
	// 	return errs.NewWithCause(fmt.Sprintf("error getting process group id for command (%d)", r.cmd.Process.Pid), err)
	// }
	// err = syscall.Kill(-pgid, 15)
	if err := syscall.Kill(-r.cmd.Process.Pid, syscall.SIGKILL); err != nil {
		return errs.NewWithCause("error killing process group", err)
	}
	if err := r.cmd.Wait(); err != nil {
		if err.Error() != "signal: killed" {
			return err
		}
	}
	return nil
}
