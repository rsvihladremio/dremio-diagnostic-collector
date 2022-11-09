/*
   Copyright 2022 Ryan SVIHLA

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

// oshelper package provides functions to wrapper os file system calls
// to better facilitate testing

package helpers

import (
	"fmt"
	"os"
	"path/filepath"
)

type Filesystem interface {
	Stat(name string) (os.FileInfo, error)
	Create(name string) (File, error)
	MkdirAll(path string, perm os.FileMode) error
	Mkdir(path string, perm os.FileMode) error
	MkdirTemp(name string, pattern string) (string, error)
	RemoveAll(path string) error
	Remove(name string) error
	WriteFile(name string, data []byte, perms os.FileMode) error
	//TempDir(dir, prefix string) (string, error)
	//TempFile(dir, prefix string) (File, error)
}

type File interface {
	Name() string
	Write(b []byte) (n int, err error)
	Sync() error
	Close() error
}

// Real file
type RealFile struct {
	file *os.File
}

// RealFileSystem wrapper
type FileSystem struct{}

//Name
func (f *RealFile) Name() string {
	return f.file.Name()
}

// Write
func (f *RealFile) Write(b []byte) (n int, err error) {
	return f.file.Write(b)
}

//Sync
func (f *RealFile) Sync() error {
	return f.file.Sync()
}

// Close
func (f *RealFile) Close() error {
	return f.file.Close()
}

// Stat
func (f FileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// Create
func (f FileSystem) Create(name string) (File, error) {
	fd, err := os.Create(filepath.Clean(name))
	if err != nil {
		return nil, err
	}
	realFile := &RealFile{
		file: fd,
	}
	return realFile, nil
}

// Mkdir
func (f FileSystem) Mkdir(name string, perms os.FileMode) error {
	err := os.Mkdir(name, perms)
	return err
}

func (f FileSystem) MkdirTemp(name string, pattern string) (string, error) {
	dir, err := os.MkdirTemp(name, pattern)
	return dir, err
}

func (f FileSystem) MkdirAll(name string, perms os.FileMode) error {
	err := os.MkdirAll(name, perms)
	return err
}

// Remove
func (f FileSystem) Remove(path string) error {
	err := os.Remove(path)
	return err
}

// RemoveAll
func (f FileSystem) RemoveAll(path string) error {
	err := os.RemoveAll(path)
	return err
}

func (f FileSystem) WriteFile(name string, data []byte, perms os.FileMode) error {
	err := os.WriteFile(name, data, perms)
	return err
}

type FakeFile struct {
}

type FakeFileSystem struct {
}

//Name
func (f *FakeFile) Name() string {
	return "fakeFile.txt"
}

// Write
func (f *FakeFile) Write(b []byte) (n int, err error) {
	fmt.Printf("Written: %v", b)
	return 0, err
}

//Sync
func (f *FakeFile) Sync() error {
	return nil
}

// Close
func (f *FakeFile) Close() error {
	return nil
}

// Stat
func (f FakeFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// Create
func (f FakeFileSystem) Create(name string) (File, error) {
	fmt.Println("Testing create")
	return &FakeFile{}, nil
}

// Mkdir
func (f FakeFileSystem) Mkdir(name string, perms os.FileMode) error {
	err := os.Mkdir(name, perms)
	return err
}

func (f FakeFileSystem) MkdirTemp(name string, pattern string) (string, error) {
	dir, err := os.MkdirTemp(name, pattern)
	return dir, err
}

func (f FakeFileSystem) MkdirAll(name string, perms os.FileMode) error {
	err := os.MkdirAll(name, perms)
	return err
}

// Remove
func (f FakeFileSystem) Remove(path string) error {
	err := os.Remove(path)
	return err
}

// RemoveAll
func (f FakeFileSystem) RemoveAll(path string) error {
	err := os.RemoveAll(path)
	return err
}

func (f FakeFileSystem) WriteFile(name string, data []byte, perms os.FileMode) error {
	err := os.WriteFile(name, data, perms)
	return err
}
