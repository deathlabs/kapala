# TODO
- [ ] Resolve and validate the notes directory and `index.json` path during startup.
- [ ] Load the existing `index.json`, or create and save an empty index if the file does not exist.
- [ ] Scan the notes directory recursively for Markdown files.
- [ ] Compare the discovered Markdown paths with the paths stored in the index.
- [ ] Chunk and embed each Markdown file that does not have an index entry.
- [ ] Remove index entries for Markdown files that no longer exist.
- [ ] Save all startup reconciliation changes to `index.json` with an atomic write.
- [ ] Start the file watcher only after startup reconciliation finishes successfully.
- [ ] Log the startup scan results without logging note contents, embeddings, or credentials.
