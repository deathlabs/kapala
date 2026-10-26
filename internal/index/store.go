package index

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// NewStore loads or creates an index store for model.
func NewStore(path string, model string) (*Store, error) {
	var (
		data  Data
		err   error
		store *Store
	)

	data, err = load(path)
	if err != nil {
		return nil, err
	}
	if len(data.Chunks) > 0 && data.Model != model {
		return nil, fmt.Errorf("index model %q does not match configured model %q", data.Model, model)
	}
	if data.Version != 0 && data.Version != Version {
		return nil, fmt.Errorf("index version %d is not supported", data.Version)
	}

	if data.Version == 0 {
		data.Version = Version
	}
	if data.Model == "" {
		data.Model = model
	}
	if data.Chunks == nil {
		data.Chunks = []Chunk{}
	}

	store = &Store{data: data, model: model, path: path}

	return store, nil
}

// ReplaceFile replaces all chunks for path after an atomic disk update.
func (store *Store) ReplaceFile(path string, contents []string, embeddings [][]float64) error {
	var (
		candidate Data
		chunks    []Chunk
		err       error
		ordinal   int
	)

	if len(contents) != len(embeddings) {
		return fmt.Errorf("chunk count %d does not match embedding count %d", len(contents), len(embeddings))
	}

	store.mutex.Lock()
	defer store.mutex.Unlock()

	candidate = cloneData(store.data)
	candidate.Chunks = removePath(candidate.Chunks, path)
	chunks = make([]Chunk, 0, len(contents))
	for ordinal = range contents {
		chunks = append(chunks, Chunk{
			Content:   contents[ordinal],
			Embedding: append([]float64(nil), embeddings[ordinal]...),
			ID:        chunkID(path, ordinal, contents[ordinal]),
			Ordinal:   ordinal,
			Path:      path,
		})
	}

	candidate.Chunks = append(candidate.Chunks, chunks...)
	candidate.Model = store.model
	candidate.UpdatedAt = time.Now().UTC()
	candidate.Version = Version

	err = writeAtomic(store.path, candidate)
	if err != nil {
		return err
	}

	store.data = candidate

	return nil
}

// RemoveFile removes all chunks for path after an atomic disk update.
func (store *Store) RemoveFile(path string) error {
	var (
		candidate Data
		err       error
	)

	store.mutex.Lock()
	defer store.mutex.Unlock()

	candidate = cloneData(store.data)
	candidate.Chunks = removePath(candidate.Chunks, path)
	if len(candidate.Chunks) == len(store.data.Chunks) {
		return nil
	}

	candidate.UpdatedAt = time.Now().UTC()
	err = writeAtomic(store.path, candidate)
	if err != nil {
		return err
	}

	store.data = candidate

	return nil
}

// Snapshot returns a safe copy of the current index data.
func (store *Store) Snapshot() Data {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	return cloneData(store.data)
}

func chunkID(path string, ordinal int, content string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(path+"\x00"+strconv.Itoa(ordinal)+"\x00"+content)))
}

func cloneData(data Data) Data {
	var (
		chunk  Chunk
		chunks []Chunk
	)

	chunks = make([]Chunk, 0, len(data.Chunks))
	for _, chunk = range data.Chunks {
		chunk.Embedding = append([]float64(nil), chunk.Embedding...)
		chunks = append(chunks, chunk)
	}
	data.Chunks = chunks

	return data
}

func load(path string) (result Data, resultError error) {
	var (
		decoder *json.Decoder
		err     error
		file    *os.File
	)

	file, err = os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	defer func() {
		err = file.Close()
		resultError = errors.Join(resultError, err)
	}()

	decoder = json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&result)
	if err != nil {
		return result, err
	}
	err = decoder.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		if err == nil {
			return result, errors.New("index contains more than one JSON value")
		}

		return result, err
	}

	return result, nil
}

func removePath(chunks []Chunk, path string) []Chunk {
	var (
		chunk  Chunk
		result []Chunk
	)

	result = make([]Chunk, 0, len(chunks))
	for _, chunk = range chunks {
		if chunk.Path != path {
			result = append(result, chunk)
		}
	}

	return result
}

func writeAtomic(path string, data Data) (result error) {
	var (
		closeError    error
		directory     string
		encoder       *json.Encoder
		err           error
		removeError   error
		temporary     *os.File
		temporaryPath string
	)

	directory = filepath.Dir(path)
	err = os.MkdirAll(directory, 0o755)
	if err != nil {
		return err
	}

	temporary, err = os.CreateTemp(directory, ".index-*.json")
	if err != nil {
		return err
	}
	temporaryPath = temporary.Name()
	defer func() {
		if temporary != nil {
			closeError = temporary.Close()
		}
		removeError = os.Remove(temporaryPath)
		if errors.Is(removeError, os.ErrNotExist) {
			removeError = nil
		}
		result = errors.Join(result, closeError, removeError)
	}()

	err = temporary.Chmod(0o600)
	if err != nil {
		return err
	}

	encoder = json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(data)
	if err != nil {
		return err
	}

	err = temporary.Sync()
	if err != nil {
		return err
	}
	err = temporary.Close()
	if err != nil {
		return err
	}
	temporary = nil

	err = os.Rename(temporaryPath, path)
	if err != nil {
		return err
	}

	return nil
}
