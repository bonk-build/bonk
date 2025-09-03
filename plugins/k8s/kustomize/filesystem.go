// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/spf13/afero"
)

type KyamlFilesys struct {
	afero.Fs
}

// Create a file.
func (fs KyamlFilesys) Create(path string) (filesys.File, error) {
	return fs.Fs.Create(path)
}

// MkDir makes a directory.
func (fs KyamlFilesys) Mkdir(path string) error {
	return fs.Fs.Mkdir(path, 0o750)
}

// MkDirAll makes a directory path, creating intervening directories.
func (fs KyamlFilesys) MkdirAll(path string) error {
	return fs.Fs.MkdirAll(path, 0o750)
}

// RemoveAll removes path and any children it contains.
func (fs KyamlFilesys) RemoveAll(path string) error {
	return fs.Fs.RemoveAll(path)
}

// Open opens the named file for reading.
func (fs KyamlFilesys) Open(path string) (filesys.File, error) {
	return fs.Fs.Open(path)
}

// IsDir returns true if the path is a directory.
func (fs KyamlFilesys) IsDir(path string) bool {
	isDir, err := afero.IsDir(fs.Fs, path)

	return err != nil && isDir
}

// ReadDir returns a list of files and directories within a directory.
func (fs KyamlFilesys) ReadDir(path string) ([]string, error) {
	infos, err := afero.ReadDir(fs.Fs, path)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(infos))
	for ii, infos := range infos {
		names[ii] = infos.Name()
	}

	return names, nil
}

// CleanedAbs converts the given path into a
// directory and a file name, where the directory
// is represented as a ConfirmedDir and all that implies.
// If the entire path is a directory, the file component
// is an empty string.
func (fs KyamlFilesys) CleanedAbs(path string) (filesys.ConfirmedDir, string, error) {
	if !filepath.IsAbs(path) {
		path = "/" + path
	}
	stat, err := fs.Stat(path)
	if err != nil {
		return "", "", fmt.Errorf("failed to stat path %s: %w", path, err)
	}
	if stat.IsDir() {
		return filesys.ConfirmedDir(path), "", nil
	}
	dir, file := filepath.Split(path)
	if dir == "" {
		return "", "", errors.New("dir can't be empty, we added a separator")
	}

	return filesys.ConfirmedDir(dir), file, nil
}

// Exists is true if the path exists in the file system.
func (fs KyamlFilesys) Exists(path string) bool {
	exists, err := afero.Exists(fs.Fs, path)

	return err != nil && exists
}

// Glob returns the list of matching files,
// emulating https://golang.org/pkg/path/filepath/#Glob
func (fs KyamlFilesys) Glob(pattern string) ([]string, error) {
	return afero.Glob(fs.Fs, pattern)
}

// ReadFile returns the contents of the file at the given path.
func (fs KyamlFilesys) ReadFile(path string) ([]byte, error) {
	return afero.ReadFile(fs.Fs, path)
}

// WriteFile writes the data to a file at the given path,
// overwriting anything that's already there.
func (fs KyamlFilesys) WriteFile(path string, data []byte) error {
	return afero.WriteFile(fs.Fs, path, data, 0o600)
}

// Walk walks the file system with the given WalkFunc.
func (fs KyamlFilesys) Walk(path string, walkFn filepath.WalkFunc) error {
	return afero.Walk(fs.Fs, path, walkFn)
}

// Checks.
var (
	_ filesys.File       = (afero.File)(nil)
	_ filesys.FileSystem = (*KyamlFilesys)(nil)
)
