// Copyright (c) 2016 Tigera, Inc. All rights reserved.

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

package selector_test

import (
	. "github.com/projectcalico/calico-go/labels/selectors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token", func() {
	It("should tokenize", func() {
		Expect(Tokenize(`a=="b"`)).To(Equal([]Token{
			{TokLabel, "a"},
			{TokEq, nil},
			{TokStringLiteral, "b"},
			{TokEof, nil},
		}))
		Expect(Tokenize(`label == "value"`)).To(Equal([]Token{
			{TokLabel, "label"},
			{TokEq, nil},
			{TokStringLiteral, "value"},
			{TokEof, nil},
		}))
		Expect(Tokenize(`a not in "bar" && !has(foo) || b in c`)).To(Equal([]Token{
			{TokLabel, "a"},
			{TokNotIn, nil},
			{TokStringLiteral, "bar"},
			{TokAnd, nil},
			{TokNot, nil},
			{TokHas, "foo"},
			{TokOr, nil},
			{TokLabel, "b"},
			{TokIn, nil},
			{TokLabel, "c"},
			{TokEof, nil},
		}))
		Expect(Tokenize(`a  not  in  "bar"  &&  ! has( foo )  ||  b  in  c `)).To(Equal([]Token{
			{TokLabel, "a"},
			{TokNotIn, nil},
			{TokStringLiteral, "bar"},
			{TokAnd, nil},
			{TokNot, nil},
			{TokHas, "foo"},
			{TokOr, nil},
			{TokLabel, "b"},
			{TokIn, nil},
			{TokLabel, "c"},
			{TokEof, nil},
		}))
		Expect(Tokenize(`a notin"bar"&&!has(foo)||b in"c"`)).To(Equal([]Token{
			{TokLabel, "a"},
			{TokNotIn, nil},
			{TokStringLiteral, "bar"},
			{TokAnd, nil},
			{TokNot, nil},
			{TokHas, "foo"},
			{TokOr, nil},
			{TokLabel, "b"},
			{TokIn, nil},
			{TokStringLiteral, "c"},
			{TokEof, nil},
		}))
	})
})
