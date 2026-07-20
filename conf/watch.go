package conf

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

func (p *Conf) Watch(filePath, xDnsPath string, sDnsPath string, reload func()) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("new watcher error: %s", err)
	}
	var reloadMu sync.Mutex
	go func() {
		var pre time.Time
		defer watcher.Close()
		for {
			select {
			case e := <-watcher.Events:
				if e.Has(fsnotify.Chmod) {
					continue
				}
				if pre.Add(10 * time.Second).After(time.Now()) {
					continue
				}
				pre = time.Now()
				eventName := e.Name
				go func() {
					time.Sleep(5 * time.Second)
					reloadMu.Lock()
					defer reloadMu.Unlock()
					switch filepath.Base(strings.TrimSuffix(eventName, "~")) {
					case filepath.Base(xDnsPath), filepath.Base(sDnsPath):
						log.Println("DNS file changed, reloading...")
					default:
						log.Println("config file changed, reloading...")
					}
					*p = *New()
					err := p.LoadFromPath(filePath)
					if err != nil {
						log.Printf("reload config error: %s", err)
						return
					}
					reload()
					log.Println("reload config success")
				}()
			case err := <-watcher.Errors:
				if err != nil {
					log.Printf("File watcher error: %s", err)
				}
			}
		}
	}()
	err = watcher.Add(filePath)
	if err != nil {
		return fmt.Errorf("watch file error: %s", err)
	}
	if xDnsPath != "" {
		err = watcher.Add(xDnsPath)
		if err != nil {
			return fmt.Errorf("watch dns file error: %s", err)
		}
	}
	if sDnsPath != "" {
		err = watcher.Add(sDnsPath)
		if err != nil {
			return fmt.Errorf("watch dns file error: %s", err)
		}
	}
	return nil
}
