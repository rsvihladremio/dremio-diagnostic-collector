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

import "os"

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
	return f.Close()
}

// Stat
func (f FileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// Create
func (f FileSystem) Create(name string) (File, error) {
	fd, err := os.Create(name)
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

// Temptdir
/*
func (fs FileSystem) TempDir(dir string, pattern string) (string, error) {
	path, err := os.MkdirTemp(dir, pattern)
	return path, err
}
*/
