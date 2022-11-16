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

// hchelper package provides functions convert collected data to a format
// thats compatile with the health check tool

/*
The typicaly DIR structue for the HC tool looks like this

20221110-141414-DDC - ?
├── configuration
│   ├── dremio-executor-0 -- 1.2.3.4-C
│   ├── dremio-executor-1 -- 12.3.45-E
│   ├── dremio-executor-2
│   └── dremio-master-0
├── dremio-cloner
├── job-profiles
├── kubernetes
├── kvstore
├── logs
│   ├── dremio-executor-0
│   ├── dremio-executor-1
│   ├── dremio-executor-2
│   └── dremio-master-0
├── node-info
│   ├── dremio-executor-0
│   ├── dremio-executor-1
│   ├── dremio-executor-2
│   └── dremio-master-0
├── queries
├── query-analyzer
│   ├── chunks
│   ├── errorchunks
│   ├── errormessages
│   └── results
└── system-tables
*/

package helpers

import (
	"io/fs"
	"path/filepath"
	"time"
)

/*
Setup the dir structure to mirror the HC tool, we do need to know all the nodes / pods
before we call this function so we can construct the right directories
*/
func SetupDirs(basedir string, nodes []string) error {
	var dir string
	var err error
	var ts string
	var perms fs.FileMode = 755
	ts = time.Now().Format("20060102_150405")
	dir = filepath.Join(basedir, ts)
	err = DDCfs.MkdirAll(dir, perms.Perm())
	if err != nil {
		return err
	}

	for _, node := range nodes {
		dir := filepath.Join(basedir, node, "configuration")
		err = DDCfs.MkdirAll(dir, perms.Perm())
		if err != nil {
			return err
		}
		dir = filepath.Join(basedir, node, "logs")
		err = DDCfs.MkdirAll(dir, perms.Perm())
		if err != nil {
			return err
		}
		dir = filepath.Join(basedir, node, "node-info")
		err = DDCfs.MkdirAll(dir, perms.Perm())
		if err != nil {
			return err
		}
	}
	dir = filepath.Join(basedir, "dremio-cloner")
	err = DDCfs.MkdirAll(dir, perms.Perm())
	if err != nil {
		return err
	}
	dir = filepath.Join(basedir, "job-profiles")
	err = DDCfs.MkdirAll(dir, perms.Perm())
	if err != nil {
		return err
	}
	dir = filepath.Join(basedir, "kubernetes")
	err = DDCfs.MkdirAll(dir, perms.Perm())
	if err != nil {
		return err
	}
	dir = filepath.Join(basedir, "kv-store")
	err = DDCfs.MkdirAll(dir, perms.Perm())
	if err != nil {
		return err
	}
	dir = filepath.Join(basedir, "queries")
	err = DDCfs.MkdirAll(dir, perms.Perm())
	if err != nil {
		return err
	}
	dir = filepath.Join(basedir, "query-analyzer/chunks")
	err = DDCfs.MkdirAll(dir, perms.Perm())
	if err != nil {
		return err
	}
	dir = filepath.Join(basedir, "query-analyzer/errorchunks")
	err = DDCfs.MkdirAll(dir, perms.Perm())
	if err != nil {
		return err
	}
	dir = filepath.Join(basedir, "query-analyzer/errormessages")
	err = DDCfs.MkdirAll(dir, perms.Perm())
	if err != nil {
		return err
	}
	dir = filepath.Join(basedir, "query-analyzer/results")
	err = DDCfs.MkdirAll(dir, perms.Perm())
	if err != nil {
		return err
	}
	dir = filepath.Join(basedir, "system-tables")
	err = DDCfs.MkdirAll(dir, perms.Perm())
	if err != nil {
		return err
	}
	return nil
}
