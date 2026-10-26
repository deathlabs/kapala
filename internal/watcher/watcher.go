// Package watcher monitors a directory tree for Markdown file changes.
package watcher

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	debounceDuration = 250 * time.Millisecond
	pollDuration     = 50 * time.Millisecond
)

type pendingEvent struct {
	event     Event
	processAt time.Time
}

// Watch monitors root recursively and passes debounced Markdown changes to handler.
func Watch(ctx context.Context, root string, handler Handler) (result error) {
	var (
		err             error
		filesystem      *fsnotify.Watcher
		filesystemEvent fsnotify.Event
		information     fs.FileInfo
		now             time.Time
		ok              bool
		pending         map[string]pendingEvent
		rootAbsolute    string
		ticker          *time.Ticker
	)

	if handler == nil {
		return errors.New("file change handler is nil")
	}

	rootAbsolute, err = filepath.Abs(root)
	if err != nil {
		return err
	}

	information, err = os.Stat(rootAbsolute)
	if err != nil {
		return err
	}
	if !information.IsDir() {
		return fmt.Errorf("notes path is not a directory: %s", rootAbsolute)
	}

	filesystem, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() {
		err = filesystem.Close()
		result = errors.Join(result, err)
	}()

	err = addDirectories(filesystem, rootAbsolute)
	if err != nil {
		return err
	}

	pending = make(map[string]pendingEvent)
	ticker = time.NewTicker(pollDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err = <-filesystem.Errors:
			if err == nil {
				return errors.New("file watcher error channel closed")
			}

			return err
		case filesystemEvent, ok = <-filesystem.Events:
			if !ok {
				return errors.New("file watcher event channel closed")
			}

			err = recordEvent(filesystem, pending, filesystemEvent)
			if err != nil {
				return err
			}
		case now = <-ticker.C:
			err = processPending(pending, now, handler)
			if err != nil {
				return err
			}
		}
	}
}

func addDirectories(filesystem *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}

		return filesystem.Add(path)
	})
}

func eventOperation(operation fsnotify.Op) (Operation, bool) {
	if operation&fsnotify.Remove != 0 {
		return OperationRemove, true
	}
	if operation&fsnotify.Rename != 0 {
		return OperationRename, true
	}
	if operation&fsnotify.Create != 0 {
		return OperationCreate, true
	}
	if operation&fsnotify.Write != 0 {
		return OperationWrite, true
	}

	return "", false
}

func isMarkdown(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".md")
}

func mergeOperation(current Operation, next Operation) Operation {
	if current == OperationCreate && next == OperationWrite {
		return current
	}

	return next
}

func processPending(pending map[string]pendingEvent, now time.Time, handler Handler) error {
	var (
		err   error
		path  string
		value pendingEvent
	)

	for path, value = range pending {
		if now.Before(value.processAt) {
			continue
		}

		err = handler(value.event)
		if err != nil {
			return err
		}

		delete(pending, path)
	}

	return nil
}
func recordEvent(filesystem *fsnotify.Watcher, pending map[string]pendingEvent, filesystemEvent fsnotify.Event) error {
	var (
		absolutePath string
		err          error
		information  fs.FileInfo
		isChange     bool
		operation    Operation
		previous     pendingEvent
	)

	absolutePath, err = filepath.Abs(filesystemEvent.Name)
	if err != nil {
		return err
	}

	if filesystemEvent.Op&fsnotify.Create != 0 {
		information, err = os.Stat(absolutePath)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err == nil && information.IsDir() {
			return addDirectories(filesystem, absolutePath)
		}
	}

	if !isMarkdown(absolutePath) {
		return nil
	}

	operation, isChange = eventOperation(filesystemEvent.Op)
	if !isChange {
		return nil
	}

	previous = pending[absolutePath]
	if previous.event.Path != "" {
		operation = mergeOperation(previous.event.Operation, operation)
	}

	pending[absolutePath] = pendingEvent{
		event: Event{
			Operation: operation,
			Path:      filepath.Clean(absolutePath),
		},
		processAt: time.Now().Add(debounceDuration),
	}

	return nil
}
