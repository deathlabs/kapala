package internal

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deathlabs/kapala/internal/index"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultSearchResultCount = 5
	maximumSearchResultCount = 50
)

var errRetrievalNotConfigured = errors.New("semantic retrieval is not configured; start kapala with --notes-dir")

func (tools *retrievalTools) semanticSearchHandler(ctx context.Context, _ *mcp.CallToolRequest, input semanticSearchInput) (*mcp.CallToolResult, semanticSearchOutput, error) {
	var (
		embedding [][]float64
		err       error
		output    semanticSearchOutput
		query     string
		topK      int
	)

	err = tools.validateConfiguration()
	if err != nil {
		return nil, output, err
	}

	query = strings.TrimSpace(input.Query)
	if query == "" {
		return nil, output, errors.New("semantic search query is empty")
	}

	topK, err = validatedTopK(input.TopK)
	if err != nil {
		return nil, output, err
	}

	embedding, err = tools.client.Embed(ctx, []string{query})
	if err != nil {
		return nil, output, err
	}

	output.Results, err = searchIndex(tools.store.Snapshot(), embedding[0], topK)
	if err != nil {
		return nil, output, err
	}

	return nil, output, nil
}

func (tools *retrievalTools) listIndexedFilesHandler(_ context.Context, _ *mcp.CallToolRequest, _ listIndexedFilesInput) (*mcp.CallToolResult, listIndexedFilesOutput, error) {
	var (
		chunk       index.Chunk
		files       []string
		indexedPath string
		output      listIndexedFilesOutput
		paths       map[string]struct{}
	)

	if tools.store == nil {
		return nil, output, errRetrievalNotConfigured
	}

	paths = make(map[string]struct{})
	for _, chunk = range tools.store.Snapshot().Chunks {
		paths[chunk.Path] = struct{}{}
	}
	files = make([]string, 0, len(paths))
	for indexedPath = range paths {
		files = append(files, indexedPath)
	}
	sort.Strings(files)
	output.Files = files

	return nil, output, nil
}

func (tools *retrievalTools) getFileContentHandler(_ context.Context, _ *mcp.CallToolRequest, input getFileContentInput) (*mcp.CallToolResult, getFileContentOutput, error) {
	var (
		content      []byte
		err          error
		filePath     string
		normalized   string
		output       getFileContentOutput
		resolvedPath string
		resolvedRoot string
	)

	err = tools.validateConfiguration()
	if err != nil {
		return nil, output, err
	}

	normalized, err = tools.validateIndexedPath(input.Path)
	if err != nil {
		return nil, output, err
	}

	resolvedRoot, err = filepath.EvalSymlinks(tools.notesRoot)
	if err != nil {
		return nil, output, err
	}
	filePath = filepath.Join(resolvedRoot, filepath.FromSlash(normalized))
	resolvedPath, err = filepath.EvalSymlinks(filePath)
	if err != nil {
		return nil, output, err
	}
	err = ensureContained(resolvedRoot, resolvedPath)
	if err != nil {
		return nil, output, err
	}

	content, err = os.ReadFile(resolvedPath)
	if err != nil {
		return nil, output, err
	}

	output = getFileContentOutput{Content: string(content), Path: normalized}

	return nil, output, nil
}

func (tools *retrievalTools) validateConfiguration() error {
	if tools.client == nil || tools.store == nil || tools.notesRoot == "" {
		return errRetrievalNotConfigured
	}

	return nil
}

func (tools *retrievalTools) validateIndexedPath(candidate string) (string, error) {
	var (
		chunk      index.Chunk
		indexed    bool
		normalized string
	)

	if candidate == "" || strings.TrimSpace(candidate) != candidate {
		return "", errors.New("indexed path is empty or contains surrounding whitespace")
	}
	if filepath.IsAbs(candidate) || path.IsAbs(candidate) || strings.Contains(candidate, `\`) {
		return "", errors.New("indexed path must be a normalized relative path")
	}

	normalized = path.Clean(candidate)
	if normalized != candidate || normalized == "." || strings.HasPrefix(normalized, "../") {
		return "", errors.New("indexed path must be a normalized relative path")
	}
	if !strings.EqualFold(path.Ext(normalized), ".md") {
		return "", errors.New("indexed path must identify a Markdown file")
	}

	for _, chunk = range tools.store.Snapshot().Chunks {
		if chunk.Path == normalized {
			indexed = true
			break
		}
	}
	if !indexed {
		return "", fmt.Errorf("path %q is not indexed", normalized)
	}

	return normalized, nil
}

func cosineSimilarity(left []float64, right []float64) (float64, error) {
	var (
		dot       float64
		index     int
		leftNorm  float64
		rightNorm float64
	)

	if len(left) != len(right) {
		return 0, fmt.Errorf("embedding dimension %d does not match index dimension %d", len(left), len(right))
	}

	for index = range left {
		dot += left[index] * right[index]
		leftNorm += left[index] * left[index]
		rightNorm += right[index] * right[index]
	}
	if leftNorm == 0 {
		return 0, errors.New("query embedding has zero magnitude")
	}
	if rightNorm == 0 {
		return 0, errors.New("indexed embedding has zero magnitude")
	}

	return dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm)), nil
}

func ensureContained(root string, candidate string) error {
	var (
		err      error
		relative string
	)

	relative, err = filepath.Rel(root, candidate)
	if err != nil {
		return err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("indexed path resolves outside the notes directory")
	}

	return nil
}

func searchIndex(data index.Data, query []float64, topK int) ([]semanticSearchResult, error) {
	var (
		chunk   index.Chunk
		err     error
		limit   int
		results []semanticSearchResult
		score   float64
		scored  []scoredChunk
		value   scoredChunk
	)

	if vectorMagnitude(query) == 0 {
		return nil, errors.New("query embedding has zero magnitude")
	}

	scored = make([]scoredChunk, 0, len(data.Chunks))
	for _, chunk = range data.Chunks {
		score, err = cosineSimilarity(query, chunk.Embedding)
		if err != nil {
			return nil, err
		}
		scored = append(scored, scoredChunk{chunk: chunk, score: score})
	}
	sort.Slice(scored, func(left int, right int) bool {
		if scored[left].score != scored[right].score {
			return scored[left].score > scored[right].score
		}
		if scored[left].chunk.Path != scored[right].chunk.Path {
			return scored[left].chunk.Path < scored[right].chunk.Path
		}
		if scored[left].chunk.Ordinal != scored[right].chunk.Ordinal {
			return scored[left].chunk.Ordinal < scored[right].chunk.Ordinal
		}

		return scored[left].chunk.ID < scored[right].chunk.ID
	})

	limit = topK
	if limit > len(scored) {
		limit = len(scored)
	}
	results = make([]semanticSearchResult, 0, limit)
	for _, value = range scored[:limit] {
		results = append(results, semanticSearchResult{
			Content: value.chunk.Content,
			Ordinal: value.chunk.Ordinal,
			Path:    value.chunk.Path,
			Score:   value.score,
		})
	}

	return results, nil
}

func validatedTopK(value int) (int, error) {
	if value == 0 {
		return defaultSearchResultCount, nil
	}
	if value < 0 {
		return 0, errors.New("top_k must be greater than zero")
	}
	if value > maximumSearchResultCount {
		return 0, fmt.Errorf("top_k cannot exceed %d", maximumSearchResultCount)
	}

	return value, nil
}

func vectorMagnitude(vector []float64) float64 {
	var (
		index     int
		magnitude float64
	)

	for index = range vector {
		magnitude += vector[index] * vector[index]
	}

	return math.Sqrt(magnitude)
}
