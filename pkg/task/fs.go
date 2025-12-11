package task

type FileSystemType uint8

const (
	FsSource FileSystemType = iota
	FsOutput
)

type FileReference struct {
	FileSystem FileSystemType `json:"fileSystem"`
	Path       string         `json:"path"`
}

// Helpers for FileReference
func SourceFile(path string) FileReference {
	return FileReference{
		FileSystem: FsSource,
		Path:       path,
	}
}
