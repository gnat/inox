package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrint(t *testing.T) {

	testCases := []string{
		"manifest {}",
		"manifest {",
		"manifest",
		"manifest ",
		//
		"includable-file",
		"includable-file {",
		"includable-file {}",
		//simple literals
		"1",
		" 1",
		`""`,
		`"1"`,
		`"\n"`,
		`"\u"`, //invalid unicode sequence
		"1x",
		"1x/s",
		"/a",
		"/a:",
		"/a:b",
		"/`[]`",
		"/`[",
		"/`[]",
		"/`[]\na",
		"https://example.com",
		"https://example.com/",
		"-x",
		"--x",
		"--name",

		"1.0..2.0",
		"1.0..",
		"1..$a",

		"1..2",
		"1..",
		"1..$a",

		"'a'..'b'",
		"'a'..",
		"1x..2x",
		"1x..2",
		"1x..",
		"1x..$a",
		//upper-bound range expression
		"..1",
		"..12",
		".../",
		"..../",
		"...../",
		//path expressions
		"/`[{x}]`",
		"/`[{x}",
		"/`[{x}]",
		"/`[{x}]\na",
		//url expressions
		"https://{host}/",
		"https://example.com/{x}",
		"https://example.com/{",
		"https://example.com/{\n",
		"https://example.com/{x",
		"https://example.com/?x={1}",
		"https://example.com/?x={",
		"https://example.com/?x={\n",
		"https://example.com/?x={1",
		"https://example.com/?x={1}&",
		"https://example.com/?x={1}&&",
		"https://example.com/?x={1}&y=2",
		"https://example.com/?x={1}&&y=2",
		"https://example.com/?x={1}&=&y=2",
		"@site/",
		//date literals
		"2020y-5mt-UTC",
		"2020y-5mt-06d-UTC",
		"2020y-5mt",
		"#a",
		//option expression
		"-x=1",
		"--x=1",
		//variable
		"(f)",
		"a",
		"a?",
		//local variable declaration
		"var",
		"var ()",
		"var a",
		"var a =",
		"var a = 1",
		"var a int = 1",
		"var (a int = 1)",
		"var (a int = 1",
		"var a namespace.pattern = 1",
		"var a {} = 1",
		"var a #{} = 1",
		//global variable declaration
		"globalvar",
		"globalvar ()",
		"globalvar a",
		"globalvar a =",
		"globalvar a = 1",
		"globalvar a int = 1",
		"globalvar (a int = 1)",
		"globalvar (a int = 1",
		"globalvar a namespace.pattern = 1",
		"globalvar a {} = 1",
		"globalvar a #{} = 1",
		//assignment
		"a = 0",
		"assign a b = c",
		//global constant declarations
		"const",
		"const (",
		"const ()",
		"const (\n)",
		"const (a = 1)",
		"const (\na = 1)",
		"const (\na = 1\n)",
		"const (\na = 1\nb= 2\n)",
		// member expression
		"a.b",
		"a.b.",
		"a.b?",
		"$a.b",
		"$a.?b",
		"$a.b.",
		"$a.b.?",
		"a.<b",
		"a.<?b",
		"a.<?",
		//double-colon expression
		"a::b",
		"a::bc",
		"a::",
		"a::1",
		"a::b::c",
		"a::b::",
		"a::b::1",
		//object
		"{}",
		"{ }",
		"{",
		"{,",
		"{,}",
		`{"a":1}`,
		`{"a" :1}`,
		`{"a": 1}`,
		`{a:1}`,
		`{a :1}`,
		`{a: 1}`,
		//record
		"#{}",
		"#{ }",
		`#{"a":1}`,
		`#{"a" :1}`,
		`#{"a": 1}`,
		//dictionary
		":{}",
		":{./a:1}",
		":{./a: 1}",
		":{./a: 1",
		":{./a: }",
		":{./a: ",
		":{./a }",
		":{a}",
		":{s3://aa: 1}",
		":{s3://aa/: 1}",
		":{a",
		":{./a: 1, ./b: 2}",
		":{./a: 1 ./b: 2}",
		":{./a: 1, ./b: 2",
		":{./a: 1, ./b: ",
		":{./a: 1, ./b:",
		":{./a: 1, ./b",
		//call
		"f()",
		"f(1)",
		"f(1,2)",
		"f",
		"f 1",
		"f 1 2",
		"a = f(1 2)",
		//pipe
		"f 1 | g 2",
		"f 1 | g 2 | h 3",
		"a = | f 1 | g 2",
		//binary expression
		"(a + b)",
		"(a - b)",
		"(a * b)",
		"(a / b)",
		"(a < b)",
		"(a <= b)",
		"(a > b)",
		"(a >= b)",
		"(a + b)",
		"(a - b)",
		"(a * b)",
		"(a / b)",
		"(a < b)",
		"(a <= b)",
		"(a > b)",
		"(a >= b)",
		"(a == b)",
		"(a is b)",
		"(a is-not b)",
		"(a in b)",
		"(a not-in b)",
		"(a keyof b)",
		"(a ulrof b)",
		"(a match b)",
		"(a match {a: | 1 | 2})",
		"(a not-match b)",
		"(a not-match {a: | 1 | 2})",
		"(a < b or c < d)",
		"(a < b or c < d",
		"(a < b or c <",
		"(a < b or c",
		"(a < b or",
		"(a < b or)",
		"(a < b or c)",
		"(a or b or c < d)",
		"(a,d)",
		"(a, d)",
		"(a ,d)",
		"(a , d)",
		//concatenations
		"concat",
		"concat \"a\"",
		"(concat)",
		"(concat \"a\")",
		"(concat \"a\"",
		"(concat \"a\" \"b\")",
		"(concat\n)",
		"(concat\n\"a\")",
		//lists
		"[]",
		"[,]",
		"[,",
		".{",
		".{,",
		".{,}",
		//patterns
		"%",
		"%a",
		"%a.",
		"%a.b",
		"%a?",
		"%a.b?",
		"%{}",
		"%{",
		"%{,",
		"%{,}",
		"%{a:1}",
		"%{a:b}",
		"%{otherprops int}",
		"%{otherprops}",
		"%{otherprops",
		"%[]",
		"%[]{}",
		"%[]%{}",
		"%[][]",
		"%[]%[]",
		"%[]a",
		"%[]%a",
		"%[1]",
		"%[1, 2]",
		"%[1]a",
		"%[1]a?",
		"%[1]a.b",
		"%[1]a.b?",
		"%str('a')",
		"%str('a'+)",
		"%str('a'=3)",
		"%str('a' 'b')",
		`%str((| "a"))`,
		`%str((| "a" | "b" ))`,
		"%``",
		"%`a`",
		"%`é`",
		"%`\n`",
		"%`\\``",
		"%`",
		"%`a",
		"%/",
		"%/a",
		"%/a:",
		"%/a:b",
		"%/...",
		"%/*",
		"%/`[a-z]`",
		"%/`[a-z]",
		"%/`[a-{end}]`",
		"%/`[a-{end}]",
		"%/{:name}",
		"%/{:name:}",
		"%/{",
		"%/{\n",
		"%/{:",
		"%/{:\n",
		"%/{:name",
		"%/{:name\n",
		"%/{name}",
		"%/{name",
		"%/{name\n",
		"%|",
		"%| 1",
		"%| 1 |",
		"%| 1 | 2",
		"%| a | b",
		"%fn()",
		"%fn() %int",
		"%fn() %int {",
		"%fn() %int {}",
		"%fn() int",
		"%fn(a int)",
		"%fn(a int) int",
		"%fn(a int) int {}",
		"%fn(a readonly int) int {}",
		"%fn(a readonly) int {}",
		"%fn() =>",
		"%fn() => 0",
		"%fn() int => 0",
		"%fn() int =",
		"%fn() int =\n",
		"pattern p =",
		"pattern p = 1",
		"pattern p = #{}",
		"pattern p = #{a: 1}",
		"pattern p = #{a",
		"pattern p = #{a:",
		"pattern p = #{a: 1",
		"pattern p = #[]",
		"pattern p = #[1]",
		"pattern p = #[1",
		"pattern p = |",
		"pattern p = | 1 | 2",
		"pattern p = |\n",
		"pattern p = |\n1",
		"pattern p = |\n1 | 2",
		"pattern p = |1\n| 2",
		"pattern p = (|)",
		"pattern p = (| 1)",
		"pattern p = (| 1 | 2)",
		"pattern p = (\n| 1 | 2)",
		"pattern p = (\n| 1 \n| 2)",
		"pattern p = %str",
		"pattern p = %str('a')",
		"pattern p = %str('a'",
		"pattern p = %str(",
		"pattern p = str('a')",
		"pattern p = str('a'",
		"pattern p = str(",
		//string template literals
		"%p``",
		"%p`",
		"%p`${int:a}`",
		"%p`${int:a}",
		"%p`${int:a",
		"%p`${int:",
		"%p`${int",
		"%p`${",
		"%https://**",
		"%https://example.com/...",
		"%https://example.com/",
		"%https://example.com/a",
		"%https://example.com?",
		"%https://example.com/a?",
		"%https://example.com/a?x=1",
		"%https://**.example.com",
		"%-x=1",
		"%--x=1",
		"%--name=\"foo\"",
		"pattern p = -x=1",
		"pattern p = --x=1",
		"pattern p = --name=\"foo\"",
		//treedata literal
		"treedata",
		"treedata 0",
		"treedata 0 {}",
		"treedata 0 {",
		"treedata {}",
		"treedata {} {}",
		"treedata {a: 1} {}",
		"treedata 0 {",
		"treedata 0 { 0 {} }",
		"treedata 0 { 0 { }",
		"treedata 0 { 0 { ",
		"treedata 0 { 0 ",
		"treedata 0 { 0 {}, }",
		"treedata 0 { 0 {}, 1}",
		"treedata 0 { 0 {1, 2}, 3}",
		"treedata 0 { 0:1}",
		"treedata 0 { 0 :1}",
		"treedata 0 { 0: 1}",
		"treedata 0 { 0 : 1}",
		"treedata 0 { 0 : 1",
		"treedata 0 { 0: 1, 2: 3}",
		"treedata 0 { 0: 1, 2: ",
		//spawn expression
		"go {} do",
		"go nil do",
		"go {} do {}",
		"go {} do f()",
		"go {} do http.read()",
		//mapping expression
		"Mapping {}",
		"Mapping { }",
		"Mapping",
		"Mapping {",
		//switch statement
		"switch",
		"switch 1",
		"switch 1 {",
		"switch 1 {}",
		"switch 1 { 1 }",
		"switch 1 { 1 {}",
		"switch 1 { 1 {",
		"switch 1 { 1 {} 2 {}",
		"switch 1 { 1 {} 2 {} }",
		"switch 1 { 1, 2 {} }",
		"switch 1 { 1 {} 2 {} defaultcase {} }",
		"switch 1 { defaultcase { }",
		"switch 1 { defaultcase ) }",
		//match statement
		"match",
		"match 1",
		"match 1 {",
		"match 1 {}",
		"match 1 { 1 }",
		"match 1 { 1 {}",
		"match 1 { 1 {",
		"match 1 { 1 {} 2 {}",
		"match 1 { 1 {} 2 {} }",
		"match 1 { 1, 2 {} }",
		"match 1 { 1 {} 2 {} defaultcase {} }",
		"match 1 { defaultcase { }",
		"match 1 { defaultcase ) }",
		//function expressions
		"fn(){}",
		"fn(arg){}",
		"fn(arg %int){}",
		"fn(arg readonly %int){}",
		"fn(arg readonly){}",
		"fn() =>",
		"fn() => 0",
		"fn() int => 0",
		"fn() int =",
		"fn() int =\n",
		//xml
		"h<div></div>",
		"(<div></div>)",
		"h<div",
		"h<div/>",
		"h<div/",
		"h<div>{1}2</div>",
		"h<div\n>{1}2</div>",
		"h<script></script>",
		"h<script>{1}2</script>",
		"h<script></",
		"h<script><",
		"h<style></style>",
		"h<style>{1}2</style>",
		"h<style></",
		"h<style><",
		"h<div>1{2}</div>",
		"h<div>1{2}3</div>",
		"h<div>{\n1}2</div>",
		"h<div>{1\n}2</div>",
		"h<div>{\n1\n}2</div>",
		`h<div a="b"></div>`,
		"h<div\na=\"b\"></div>",
		`h<div a=></div>`,
		`h<div "a"="b"></div>`,
		`h<div a="b"/>`,
		`h<div a=/>`,
		`h<div "a"="b"/>`,
		"h<div></span></span></div>",
		"h<div></span>1</span>2</div>",
		"h<div {}></div>",
		"h<div {1}></div>",
		"h<div {1></div>",
		"h<div {",
		"h<script h>on click</script>",
		"h<script type=\"text/hyperscript\">on click</script>",
		"h<script type=\"text/hyperscript\" n>on click</script>",
		"h<script>f()</script>",
		//imports
		"import",
		"import res",
		"import res /a",
		"import res /a {}",
		"import /a",
		//structs
		"struct Lexer",
		"struct Lexer {}",
		"struct Lexer {index: 0}",
		//new
		"new Lexer",
		"new Lexer {}",
		"new Lexer {",
		"new Lexer {index: 0}",
		"new Lexer {index: 0",
		//pointer types
		"fn(x *x){}",
		"fn(l *Lexer){}",
		//dereference expressions
		"*x",
		"*xy",
		//others
		"@(1)",
	}

	n, _ := ParseChunk("https://example.com/?x={1}&", "")
	s := SPrint(n, n, PrintConfig{})
	assert.Equal(t, "https://example.com/?x={1}&", s)

	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			n, _ := ParseChunk(testCase, "")
			s := SPrint(n, n, PrintConfig{KeepLeadingSpace: true, KeepTrailingSpace: true})
			assert.Equal(t, testCase, s)
		})
	}

	t.Run("no space kept", func(t *testing.T) {
		n, _ := ParseChunk("a=1;b=2;c=3", "")
		s := SPrint(n.Statements[2], n, PrintConfig{})
		assert.Equal(t, "c=3", s)

		n, _ = ParseChunk("  c=3", "")
		s = SPrint(n, n, PrintConfig{})
		assert.Equal(t, "c=3", s)

		n, _ = ParseChunk("\nc=3", "")
		s = SPrint(n, n, PrintConfig{})
		assert.Equal(t, "c=3", s)
	})

	t.Run("keep leading space", func(t *testing.T) {
		n, _ := ParseChunk("a=1;b=2;c=3", "")
		s := SPrint(n.Statements[2], n, PrintConfig{KeepLeadingSpace: true})
		assert.Equal(t, "        c=3", s)

		n, _ = ParseChunk("  c=3", "")
		s = SPrint(n, n, PrintConfig{KeepLeadingSpace: true})
		assert.Equal(t, "  c=3", s)

		n, _ = ParseChunk("\nc=3", "")
		s = SPrint(n, n, PrintConfig{KeepLeadingSpace: true})
		assert.Equal(t, "\nc=3", s)

		n, _ = ParseChunk("\n c=3", "")
		s = SPrint(n, n, PrintConfig{KeepLeadingSpace: true})
		assert.Equal(t, "\n c=3", s)

		n, _ = ParseChunk(" \nc=3", "")
		s = SPrint(n, n, PrintConfig{KeepLeadingSpace: true})
		assert.Equal(t, " \nc=3", s)
	})

}
