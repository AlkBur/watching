package watching

import (
	"time"
	"path/filepath"
	"fmt"
	"os"
	"sync"
	"strings"
)

type cacheFiles struct {
	f map[string]os.FileInfo
	mux sync.RWMutex
}

type showMatches struct {
	m []string
	mux sync.RWMutex
}

type Watching struct {
	duration time.Duration
	dir string

	matches *showMatches
	files *cacheFiles

	compiler func([]os.FileInfo)

	start bool

	timeout <- chan time.Time
	done chan struct{}
}

func newCacheFiles() *cacheFiles {
	return &cacheFiles{
		f: make(map[string]os.FileInfo),
	}
}

func newShowMatches() *showMatches {
	return &showMatches{
		m: make([]string,0, 2),
	}
}

func New(compiler func([]os.FileInfo)) *Watching {
	w := &Watching{
		duration: 1000,
		dir: "./",
		matches: newShowMatches(),
		files: newCacheFiles(),
		compiler: compiler,
		start: false,

		done: make(chan struct{}),
	}
	return w
}

func (c *cacheFiles)add(f os.FileInfo)  {
	c.mux.Lock()
	c.f[f.Name()]=f
	c.mux.Unlock()
}

func (c *cacheFiles)get(name string) (os.FileInfo, bool) {
	c.mux.RLock()
	f, ok := c.f[name]
	c.mux.RUnlock()
	return f, ok
}

func (c *cacheFiles)len() (int) {
	c.mux.RLock()
	r := len(c.f)
	c.mux.RUnlock()
	return r
}

func (m *showMatches)add(name string)  {
	m.mux.Lock()
	m.m = append(m.m, name)
	m.mux.Unlock()
}

func (m *showMatches)get(i int) (string) {
	m.mux.RLock()
	if i < len(m.m) && i >=0 {
		return m.m[i]
	}
	m.mux.RUnlock()
	fmt.Println("Error i:", i)
	return ""
}

func (m *showMatches)len() (int) {
	m.mux.RLock()
	r := len(m.m)
	m.mux.RUnlock()
	return r
}

func (w *Watching)Close()  {
	w.done <- struct{}{}
}

func (w *Watching)AddWatcher(matche string)  {
	if strings.HasSuffix(matche, "*"){
		w.matches.add("./"+matche)
	}else{
		w.matches.add(matche)
	}
}

func (w *Watching)SetTimeout(t time.Duration)  {
	w.duration = t
}

func (w *Watching)Run()  {
	go w.run()
}

func (w *Watching)run()  {
	w.timeout = time.After(1 * time.Millisecond)
	for {
		select {
		case <-w.done:
			return
		case <-w.timeout:
			change_file :=  make(chan os.FileInfo)
			arr_change := make([]os.FileInfo,0)
			done := make(chan struct{})
			c := make(chan struct{})

			go func() {
				for {
					select {
					case info := <-change_file:
						arr_change = append(arr_change, info)
					case <-done:
						c <- struct {}{}
						return
					}
				}
			}()

			var wg sync.WaitGroup
			l := w.matches.len()
			for i:= 0; i < l; i++ {
				str := w.matches.get(i)
				wg.Add(1)
				go func() {
					w.checkFiles(str, change_file)
					wg.Done()
				}()
			}
			wg.Wait()
			done <- struct {}{}
			<-c
			close(change_file)

			for _, f :=range arr_change {
				w.files.add(f)
			}
			if w.start {
				fmt.Println("Отправим данные об изменении!")
				w.compiler(arr_change)
			}else{
				w.start = true
			}
			//Запуск таймера
			w.timeout = time.After(w.duration * time.Millisecond)
		}
	}
}

func (w *Watching)checkFiles(matche string, change_file chan os.FileInfo) {
	var old os.FileInfo
	var ok bool

	files, _  := filepath.Glob(matche)

	for _, f:= range files {
		info, err := os.Lstat(f)
		if err != nil {
			continue
		}
		old, ok = w.files.get(info.Name())
		if ok {
			if old.Size() != info.Size() || old.ModTime() != info.ModTime() {
				change_file <- info
			}
		}else{
			change_file <- info
		}
	}
}