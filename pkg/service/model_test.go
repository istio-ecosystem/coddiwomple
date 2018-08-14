// Copyright 2018 Tetrate Labs
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"testing"

	"k8s.io/api/core/v1"
	mv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStore_Insert(t *testing.T) {
	tests := []struct {
		name string
		in   *v1.Service
		out  string
	}{
		{"happy", &v1.Service{ObjectMeta: mv1.ObjectMeta{Name: "a", Namespace: "b"}}, "a.b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			underTest := newStore()
			underTest.Insert(tt.in)

			if _, exists := underTest.s[tt.out]; !exists {
				t.Fatalf("underTest.s[%q] = _, false expected to be found; undertest: %v", tt.out, underTest)
			}
		})
	}
}
