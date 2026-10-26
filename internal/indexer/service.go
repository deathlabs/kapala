package indexer

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/deathlabs/kapala/internal/chunking"
	"github.com/deathlabs/kapala/internal/embeddings"
	"github.com/deathlabs/kapala/internal/index"
	"github.com/deathlabs/kapala/internal/watcher"
)

// New creates a file indexing service.
func New(root string, client *embeddings.Client, store *index.Store) (*Service, error) {
	var (
		err          error
		rootAbsolute string
		service      *Service
	)

	rootAbsolute, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	service = &Service{client: client, root: rootAbsolute, store: store}

	return service, nil
}

// Handle applies one file event to the semantic index.
func (service *Service) Handle(ctx context.Context, event watcher.Event) (Result, error) {
	var (
		content    []byte
		contents   []string
		embeddings [][]float64
		err        error
		path       string
		result     Result
	)

	path, err = service.relativePath(event.Path)
	if err != nil {
		return result, err
	}
	result.Path = path

	if event.Operation == watcher.OperationRemove || event.Operation == watcher.OperationRename {
		err = service.store.RemoveFile(path)
		if err != nil {
			return result, err
		}

		return result, nil
	}

	content, err = os.ReadFile(event.Path)
	if errors.Is(err, fs.ErrNotExist) {
		err = service.store.RemoveFile(path)
		if err != nil {
			return result, err
		}

		return result, nil
	}
	if err != nil {
		return result, err
	}

	contents = chunking.Markdown(string(content))
	if len(contents) > 0 {
		embeddings, err = service.client.Embed(ctx, contents)
		if err != nil {
			return result, err
		}
	}

	err = service.store.ReplaceFile(path, contents, embeddings)
	if err != nil {
		return result, err
	}
	result.Chunks = len(contents)

	return result, nil
}

func (service *Service) relativePath(path string) (string, error) {
	var (
		err      error
		relative string
	)

	relative, err = filepath.Rel(service.root, path)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("changed file is outside the notes directory")
	}

	return filepath.ToSlash(filepath.Clean(relative)), nil
}
