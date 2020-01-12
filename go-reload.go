package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/fsnotify/fsnotify"
)

var (
	directory       string
	defaultInterval = 5
	cmdArgs         []string
	cmdEnv          []string
	cmdPath         string
)

func init() {
	flag.StringVarP(&directory, "dir", "d", "", "directory to watch")
	flag.IntVarP(&defaultInterval, "int", "i", 5, "interval to rescan the dir for new files")
	cmdArgs = os.Args
	cmdEnv = os.Environ()
	cmdPath, _ = filepath.Abs(os.Args[0])
}

func printUsage() {
	fmt.Printf("Usage: %s [options]\n", os.Args[0])
	fmt.Println("Options:")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {

	flag.Parse()
	if flag.NFlag() == 0 {
		printUsage()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	defer watcher.Close()

	if _, err := os.Stat(directory); err == nil {
		log.Println(">>> Go-reload")
		log.Printf(`Watching ".go" files in %s directory, CTRL+C to stop`, directory)
		go startWatching(watcher, directory, defaultInterval)
	} else {
		log.Fatal("directory doesnt exists : ", err)
		return
	}

	errCh := make(chan error)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				switch {
				case event.Op&fsnotify.Write == fsnotify.Write:
					log.Printf("Write: %s: %s", event.Op, event.Name)
					restart()
				case event.Op&fsnotify.Create == fsnotify.Create:
					log.Printf("Create: %s: %s", event.Op, event.Name)
					restart()
				case event.Op&fsnotify.Remove == fsnotify.Remove:
					log.Printf("Remove: %s: %s", event.Op, event.Name)
					restart()
				case event.Op&fsnotify.Rename == fsnotify.Rename:
					log.Printf("Rename: %s: %s", event.Op, event.Name)
					restart()
				case event.Op&fsnotify.Chmod == fsnotify.Chmod:
					log.Printf("Chmod: %s: %s", event.Op, event.Name)
				}
			case err := <-watcher.Errors:
				errCh <- err
			}
		}
	}()

	log.Fatalln(<-errCh)
}

func startWatching(watcher *fsnotify.Watcher, directory string, defaultInterval int) {
	done := make(chan struct{})
	go func() {
		done <- struct{}{}
	}()
	ticker := time.NewTicker(time.Duration(defaultInterval) * time.Second)
	defer ticker.Stop()

	for ; ; <-ticker.C {
		<-done
		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Fatal(err)
			}
			f_mode := info.Mode()
			if f_mode.IsDir() {
				return nil
			} else if f_mode.IsRegular() {
				if filepath.Ext(path) == ".go" {
					return watcher.Add(path)
				}
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			done <- struct{}{}
		}()
	}
}

func restart() {
	binary, err := exec.LookPath(cmdPath)
	if err != nil {
		log.Printf("Error: %s", err)
		return
	}
	time.Sleep(1 * time.Second)
	execErr := syscall.Exec(binary, cmdArgs, cmdEnv)
	if execErr != nil {
		log.Printf("error : %s %v", binary, execErr)
	}
}
