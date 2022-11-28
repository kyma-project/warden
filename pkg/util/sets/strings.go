/*
Copyright 2022.

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

package sets

import (
	"strings"
	"sync"
)

// Strings is a basic thread-safe implementation of list of unique strings
// based on builtin and fast map[string]any implementation
type Strings struct {
	mu sync.Mutex
	m  map[string]any
}

func (s *Strings) Add(val string) {
	s.mu.Lock()
	if s.m == nil {
		s.m = make(map[string]any)
	}
	// push empty struct
	s.m[val] = struct{}{}
	s.mu.Unlock()
}

func (s *Strings) List() []string {
	var res []string
	for v := range s.m {
		res = append(res, v)
	}
	return res
}

func (s *Strings) Remove(val string) {
	s.mu.Lock()
	delete(s.m, val)
	s.mu.Unlock()
}

func (s *Strings) Has(val string) bool {
	_, ok := s.m[val]
	return ok
}

func (s *Strings) String() string {
	return strings.Join(s.List(), ",")
}

func (s *Strings) Walk(walkFunc func(string)) {
	for k := range s.m {
		walkFunc(k)
	}
}
