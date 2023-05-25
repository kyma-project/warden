package config

import (
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"os"
	"path"
	"time"
)

type watcher struct {
	lastCheck time.Time
	checksum  string
}

func Watch(filePath string, log *zap.SugaredLogger) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Wrap(err, "while creating file watcher")
	}
	go func() {
		defer func() {
			watcher.Close()
		}()
		log.Debug("starting watching events")
		for {
			log.Debug("check file event")
			select {
			case event := <-watcher.Events:
				log.Debugf("event name: %s, op: %s", event.Name, event.Op)
				log.Info("Config changed, restarting")
				os.Exit(0)
			case watchErr := <-watcher.Errors:
				log.Error(watchErr.Error())
				os.Exit(1)
			}
		}
	}()

	dirPath := path.Dir(filePath)

	log.Infof("config watcher started for: %s", filePath)
	if err := watcher.Add(dirPath); err != nil {
		return errors.Wrap(err, "while adding filePath to watch")
	}
	return nil
}
