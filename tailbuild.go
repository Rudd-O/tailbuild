package main

import "bufio"
import "errors"
import "flag"
import "fmt"
import "io"
import "os"
import "os/signal"
import "path/filepath"
import "strconv"
import "strings"
import "time"

var ErrNoLogs = errors.New("no log files in supplied directory")

type Tailer struct {
	filename string
	fp       *os.File
	producer chan []byte
	stopper  chan bool
}

func (t Tailer) run() {
	now := time.After(0 * time.Second)
doneopening:
	for {
		select {
		case <-now:
			var err error
			t.fp, err = os.Open(t.filename)
			if err != nil {
				if !os.IsNotExist(err) {
					panic(err)
				}
				now = time.After(1 * time.Second)
				continue
			}
			break doneopening
		case <-t.stopper:
			close(t.producer)
			return
		}
	}

	r := bufio.NewReader(t.fp)
	now = time.After(0 * time.Second)
	var akku []byte = nil
	for {
	waitforread:
		select {
		case <-t.stopper:
			if akku != nil {
				t.producer <- append(akku, '\n')
			}
			close(t.producer)
			return
		case <-now:
			for {
				b, err := r.ReadBytes('\n')
				if err != nil {
					if err == io.EOF {
						akku = append(akku, b...)
						now = time.After(1 * time.Second)
						break waitforread
					}
					panic(err)
				}
				c := append(akku, b...)
				t.producer <- c
				akku = nil
			}
		}
	}
}

func (t Tailer) Start() {
	go t.run()
}

func (t Tailer) Stop() {
	t.stopper <- true
	close(t.stopper)
}

func (t Tailer) Producer() chan []byte {
	return t.producer
}

func NewTailer(filename string) Tailer {
	t := Tailer{
		filename: filename,
		stopper:  make(chan bool),
		producer: make(chan []byte),
	}
	return t
}

func getlatestdir(projectdir string, subdir string) (int, error) {
	buildsdir := filepath.Join(projectdir, subdir)
	dir, err := os.Open(buildsdir)
	if err != nil {
		return 0, err
	}
	fis, err := dir.Readdir(-1)
	if err != nil {
		return 0, err
	}
	var latestfi os.FileInfo
	for _, fi := range fis {
		num, err := strconv.Atoi(fi.Name())
		if err != nil || num < 0 {
			continue
		}
		if latestfi == nil {
			latestfi = fi
			continue
		}
		if fi.ModTime().After(latestfi.ModTime()) {
			latestfi = fi
		}
	}
	if latestfi == nil {
		return 0, ErrNoLogs
	}
	return strconv.Atoi(latestfi.Name())
}

// GlobEscape escapes characters *?[] so that they are
// not included in any match expressions during a Glob.
func GlobEscape(path string) string {
	magic := map[string]bool{
		"*": true,
		"?": true,
		"[": true,
		"]": true,
	}

	sects := []string{}
	split := strings.Split(path, "")
	prev := ""

	for _, strCh := range split {
		if prev != "\\" {
			if _, isMagic := magic[strCh]; isMagic {
				strCh = "[" + strCh + "]"
			}
		}

		prev = strCh
		sects = append(sects, strCh)
	}

	return strings.Join(sects, "")
}

func axisdirs(basedir string) ([]string, []error) {
	allofit := []string{}
	var errors []error
	for i := 1; i <= 10; i++ {
		dir := GlobEscape(basedir)
		for j := i; j != 0; j-- {
			dir = filepath.Join(dir, "axis-*", "*")
		}
		matches, errs := filepath.Glob(dir)
		if errs != nil {
			errors = append(errors, errs)
		}
		if len(matches) > 0 {
			allofit = append(allofit, matches...)
		} else {
			break
		}
	}
	return allofit, errors
}

func discoverlogs(projectdir string, buildnumber int, discoveraxes bool) ([]string, []error) {
	subdir := "builds"
	var errors []error
	if buildnumber < 1 {
		var err error
		buildnumber, err = getlatestdir(projectdir, subdir)
		if err != nil {
			return nil, []error{err}
		}
	}
	logfiles := []string{
		filepath.Join(projectdir, subdir, fmt.Sprintf("%d", buildnumber), "log"),
	}
	if discoveraxes {
		subsubdirs, errs := axisdirs(filepath.Join(projectdir, "configurations"))
		if len(errs) > 0 {
			errors = append(errors, errs...)
		}
		for _, subsubdir := range subsubdirs {
			sublogfiles, errs := discoverlogs(subsubdir, buildnumber, false)
			if len(errs) > 0 {
				errors = append(errors, errs...)
			}
			logfiles = append(logfiles, sublogfiles...)
		}
	}
	return logfiles, errors
}

type TailerMessage struct {
	t Tailer
	s []byte
}

type Formatter interface {
	Format(string, []byte) string
}

type PlainFormatter struct{}

func (f *PlainFormatter) Format(filename string, logline []byte) string {
	return fmt.Sprintf("%s: %s", filename, logline)
}

func NewPlainFormatter() *PlainFormatter {
	return &PlainFormatter{}
}

type ColorFormatter struct {
	colormapping map[string]int
	colorindex   int
	colorarray   []int
}

func (f *ColorFormatter) Format(filename string, logline []byte) string {
	if _, ok := f.colormapping[filename]; !ok {
		f.colormapping[filename] = f.colorindex
		f.colorindex = f.colorindex + 1
		if f.colorindex >= len(f.colorarray) {
			f.colorindex = 0
		}
	}
	thecolor := f.colorarray[f.colormapping[filename]]
	coloron := []byte{27, '[', '1', ';'}
	coloron = append(coloron, []byte(strconv.Itoa(thecolor))...)
	coloron = append(coloron, 'm')
	coloroff := []byte{27, '[', '0', 'm'}
	return fmt.Sprintf("%s%s:%s %s", coloron, filename, coloroff, logline)
}

func NewColorFormatter() *ColorFormatter {
//# Black       0;30     Dark Gray     1;30
//# Blue        0;34     Light Blue    1;34
//# Green       0;32     Light Green   1;32
//# Cyan        0;36     Light Cyan    1;36
//# Red         0;31     Light Red     1;31
//# Purple      0;35     Light Purple  1;35
//# Brown       0;33     Yellow        1;33
//# Light Gray  0;37     White         1;37

	return &ColorFormatter{make(map[string]int), 0, []int{34, 32, 36, 31, 35, 33, 37}}
}

var jobsdir string
var buildnumber int

func init() {
	flag.StringVar(&jobsdir, "jobsdir", "/var/lib/jenkins", "Jenkins job directory that contains the builds/ subdirectory")
	flag.IntVar(&buildnumber, "buildnumber", 0, "build number of logs to tail (default latest)")
}

func main() {
	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "error: you must supply a project name as the first argument")
		os.Exit(64)
	}
	if buildnumber < 0 {
		fmt.Fprintln(os.Stderr, "error: the build number must be a positive integer")
		os.Exit(64)
	}
	project := flag.Args()[0]

	jenkinsprojectdir := filepath.Join(jobsdir, project)

	logfiles, errs := discoverlogs(jenkinsprojectdir, buildnumber, true)
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}
		os.Exit(4)
	}

	formatter := NewColorFormatter()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	tailers := []Tailer{}
	multiplex := make(chan TailerMessage)
	for _, logfile := range logfiles {
		t := NewTailer(logfile)
		tailers = append(tailers, t)
		go func() {
			for {
				select {
				case chunk := <-t.Producer():
					multiplex <- TailerMessage{t, chunk}
					if chunk == nil {
						return
					}
				}
			}
		}()
	}
	for _, t := range tailers {
		t.Start()
	}

	for len(tailers) > 0 {
		select {
		case <-sig:
			for _, t := range tailers {
				t.Stop()
			}
		case tmsg := <-multiplex:
			t := tmsg.t
			bytearray := tmsg.s
			if bytearray == nil {
				newtailers := []Tailer{}
				for _, u := range tailers {
					if u != t {
						newtailers = append(newtailers, u)
					}
				}
				tailers = newtailers
			} else {
				line := formatter.Format(t.filename, bytearray)
				fmt.Printf(line)
			}
		}
	}
}
