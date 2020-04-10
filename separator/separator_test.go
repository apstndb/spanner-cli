package separator

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSeparatorSkipComments(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		str          string
		wantRemained string
	}{
		{
			desc:         "single line comment (#)",
			str:          "# SELECT 1;\n",
			wantRemained: "",
		},
		{
			desc:         "single line comment (--)",
			str:          "-- SELECT 1;\n",
			wantRemained: "",
		},
		{
			desc:         "multiline comment",
			str:          "/* SELECT\n1; */",
			wantRemained: "",
		},
		{
			desc:         "single line comment (#) and statement",
			str:          "# SELECT 1;\nSELECT 2;",
			wantRemained: "SELECT 2;",
		},
		{
			desc:         "single line comment (--) and statement",
			str:          "-- SELECT 1;\nSELECT 2;",
			wantRemained: "SELECT 2;",
		},
		{
			desc:         "multiline comment and statement",
			str:          "/* SELECT\n1; */ SELECT 2;",
			wantRemained: " SELECT 2;",
		},
		{
			desc:         "single line comment (#) not terminated",
			str:          "# SELECT 1",
			wantRemained: "",
		},
		{
			desc:         "single line comment (--) not terminated",
			str:          "-- SELECT 1",
			wantRemained: "",
		},
		{
			desc:         "multiline comment not terminated",
			str:          "/* SELECT\n1;",
			wantRemained: "",
		},
		{
			desc:         "not comments",
			str:          "SELECT 1;",
			wantRemained: "SELECT 1;",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			s := newSeparator(tt.str)
			s.skipComments()

			remained := string(s.str)
			if remained != tt.wantRemained {
				t.Errorf("consumeComments(%q) remained %q, but want = %q", tt.str, remained, tt.wantRemained)
			}
		})
	}
}

func TestSeparatorConsumeString(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		str          string
		want         string
		wantRemained string
	}{
		{
			desc:         "double quoted string",
			str:          `"test" WHERE`,
			want:         `"test"`,
			wantRemained: " WHERE",
		},
		{
			desc:         "single quoted string",
			str:          `'test' WHERE`,
			want:         `'test'`,
			wantRemained: " WHERE",
		},
		{
			desc:         "tripled quoted string",
			str:          `"""test""" WHERE`,
			want:         `"""test"""`,
			wantRemained: " WHERE",
		},
		{
			desc:         "quoted string with escape sequence",
			str:          `"te\"st" WHERE`,
			want:         `"te\"st"`,
			wantRemained: " WHERE",
		},
		{
			desc:         "double quoted empty string",
			str:          `"" WHERE`,
			want:         `""`,
			wantRemained: " WHERE",
		},
		{
			desc:         "tripled quoted string with new line",
			str:          "'''t\ne\ns\nt''' WHERE",
			want:         "'''t\ne\ns\nt'''",
			wantRemained: " WHERE",
		},
		{
			desc:         "triple quoted empty string",
			str:          `"""""" WHERE`,
			want:         `""""""`,
			wantRemained: " WHERE",
		},
		{
			desc:         "multi-byte character in string",
			str:          `"テスト" WHERE`,
			want:         `"テスト"`,
			wantRemained: " WHERE",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			s := newSeparator(tt.str)
			s.consumeString()

			got := s.sb.String()
			if got != tt.want {
				t.Errorf("consumeString(%q) = %q, but want = %q", tt.str, got, tt.want)
			}

			remained := string(s.str)
			if remained != tt.wantRemained {
				t.Errorf("consumeString(%q) remained %q, but want = %q", tt.str, remained, tt.wantRemained)
			}
		})
	}
}

func TestSeparatorConsumeRawString(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		str          string
		want         string
		wantRemained string
	}{
		{
			desc:         "raw string (r)",
			str:          `r"test" WHERE`,
			want:         `r"test"`,
			wantRemained: " WHERE",
		},
		{
			desc:         "raw string (R)",
			str:          `R'test' WHERE`,
			want:         `R'test'`,
			wantRemained: " WHERE",
		},
		{
			desc:         "raw string with escape sequence",
			str:          `r"test\abc" WHERE`,
			want:         `r"test\abc"`,
			wantRemained: " WHERE",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			s := newSeparator(tt.str)
			s.consumeRawString()

			got := s.sb.String()
			if got != tt.want {
				t.Errorf("consumeRawString(%q) = %q, but want = %q", tt.str, got, tt.want)
			}

			remained := string(s.str)
			if remained != tt.wantRemained {
				t.Errorf("consumeRawString(%q) remained %q, but want = %q", tt.str, remained, tt.wantRemained)
			}
		})
	}
}

func TestSeparatorConsumeBytesString(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		str          string
		want         string
		wantRemained string
	}{
		{
			desc:         "bytes string (b)",
			str:          `b"test" WHERE`,
			want:         `b"test"`,
			wantRemained: " WHERE",
		},
		{
			desc:         "bytes string (B)",
			str:          `B'test' WHERE`,
			want:         `B'test'`,
			wantRemained: " WHERE",
		},
		{
			desc:         "bytes string with hex escape",
			str:          `b"\x12\x34\x56" WHERE`,
			want:         `b"\x12\x34\x56"`,
			wantRemained: " WHERE",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			s := newSeparator(tt.str)
			s.consumeBytesString()

			got := s.sb.String()
			if got != tt.want {
				t.Errorf("consumeBytesString(%q) = %q, but want = %q", tt.str, got, tt.want)
			}

			remained := string(s.str)
			if remained != tt.wantRemained {
				t.Errorf("consumeBytesString(%q) remained %q, but want = %q", tt.str, remained, tt.wantRemained)
			}
		})
	}
}

func TestSeparatorConsumeRawBytesString(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		str          string
		want         string
		wantRemained string
	}{
		{
			desc:         "raw bytes string (rb)",
			str:          `rb"test" WHERE`,
			want:         `rb"test"`,
			wantRemained: " WHERE",
		},
		{
			desc:         "raw bytes string (RB)",
			str:          `RB"test" WHERE`,
			want:         `RB"test"`,
			wantRemained: " WHERE",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			s := newSeparator(tt.str)
			s.consumeRawBytesString()

			got := s.sb.String()
			if got != tt.want {
				t.Errorf("consumeRawBytesString(%q) = %q, but want = %q", tt.str, got, tt.want)
			}

			remained := string(s.str)
			if remained != tt.wantRemained {
				t.Errorf("consumeRawBytesString(%q) remained %q, but want = %q", tt.str, remained, tt.wantRemained)
			}
		})
	}
}

func TestSeparateInput(t *testing.T) {
	for _, tt := range []struct {
		desc  string
		input string
		want  []InputStatement
	}{
		{
			desc:  "single query",
			input: `SELECT "123";`,
			want: []InputStatement{
				{
					Statement: `SELECT "123"`,
					Delim:     DelimiterHorizontal,
				},
			},
		},
		{
			desc:  "double queries",
			input: `SELECT "123"; SELECT "456";`,
			want: []InputStatement{
				{
					Statement: `SELECT "123"`,
					Delim:     DelimiterHorizontal,
				},
				{
					Statement: `SELECT "456"`,
					Delim:     DelimiterHorizontal,
				},
			},
		},
		{
			desc:  "quoted identifier",
			input: "SELECT `1`, `2`; SELECT `3`, `4`;",
			want: []InputStatement{
				{
					Statement: "SELECT `1`, `2`",
					Delim:     DelimiterHorizontal,
				},
				{
					Statement: "SELECT `3`, `4`",
					Delim:     DelimiterHorizontal,
				},
			},
		},
		{
			desc:  "vertical delim",
			input: `SELECT "123"\G`,
			want: []InputStatement{
				{
					Statement: `SELECT "123"`,
					Delim:     DelimiterVertical,
				},
			},
		},
		{
			desc:  "mixed delim",
			input: `SELECT "123"; SELECT "456"\G SELECT "789";`,
			want: []InputStatement{
				{
					Statement: `SELECT "123"`,
					Delim:     DelimiterHorizontal,
				},
				{
					Statement: `SELECT "456"`,
					Delim:     DelimiterVertical,
				},
				{
					Statement: `SELECT "789"`,
					Delim:     DelimiterHorizontal,
				},
			},
		},
		{
			desc:  "sql query",
			input: `SELECT * FROM t1 WHERE id = "123" AND "456"; DELETE FROM t2 WHERE true;`,
			want: []InputStatement{
				{
					Statement: `SELECT * FROM t1 WHERE id = "123" AND "456"`,
					Delim:     DelimiterHorizontal,
				},
				{
					Statement: `DELETE FROM t2 WHERE true`,
					Delim:     DelimiterHorizontal,
				},
			},
		},
		{
			desc:  "second query is empty",
			input: `SELECT 1; ;`,
			want: []InputStatement{
				{
					Statement: `SELECT 1`,
					Delim:     DelimiterHorizontal,
				},
				{
					Statement: ``,
					Delim:     DelimiterHorizontal,
				},
			},
		},
		{
			desc:  "new line just after delim",
			input: "SELECT 1;\n SELECT 2\\G\n",
			want: []InputStatement{
				{
					Statement: `SELECT 1`,
					Delim:     DelimiterHorizontal,
				},
				{
					Statement: `SELECT 2`,
					Delim:     DelimiterVertical,
				},
			},
		},
		{
			desc:  "horizontal delimiter in string",
			input: `SELECT "1;2;3"; SELECT 'TL;DR';`,
			want: []InputStatement{
				{
					Statement: `SELECT "1;2;3"`,
					Delim:     DelimiterHorizontal,
				},
				{
					Statement: `SELECT 'TL;DR'`,
					Delim:     DelimiterHorizontal,
				},
			},
		},
		{
			desc:  `vertical delimiter in string`,
			input: `SELECT r"1\G2\G3"\G SELECT r'4\G5\G6'\G`,
			want: []InputStatement{
				{
					Statement: `SELECT r"1\G2\G3"`,
					Delim:     DelimiterVertical,
				},
				{
					Statement: `SELECT r'4\G5\G6'`,
					Delim:     DelimiterVertical,
				},
			},
		},
		{
			desc:  "delimiter in quoted identifier",
			input: "SELECT `1;2`; SELECT `3;4`;",
			want: []InputStatement{
				{
					Statement: "SELECT `1;2`",
					Delim:     DelimiterHorizontal,
				},
				{
					Statement: "SELECT `3;4`",
					Delim:     DelimiterHorizontal,
				},
			},
		},
		{
			desc:  `query has new line just before delimiter`,
			input: "SELECT '123'\n; SELECT '456'\n\\G",
			want: []InputStatement{
				{
					Statement: `SELECT '123'`,
					Delim:     DelimiterHorizontal,
				},
				{
					Statement: `SELECT '456'`,
					Delim:     DelimiterVertical,
				},
			},
		},
		{
			desc:  `DDL`,
			input: "CREATE t1 (\nId INT64 NOT NULL\n) PRIMARY KEY (Id);",
			want: []InputStatement{
				{
					Statement: "CREATE t1 (\nId INT64 NOT NULL\n) PRIMARY KEY (Id)",
					Delim:     DelimiterHorizontal,
				},
			},
		},
		{
			desc:  `statement with multiple comments`,
			input: "# comment;\nSELECT /* comment */ 1; --comment\nSELECT 2;/* comment */",
			want: []InputStatement{
				{
					Statement: "SELECT  1",
					Delim:     DelimiterHorizontal,
				},
				{
					Statement: "SELECT 2",
					Delim:     DelimiterHorizontal,
				},
			},
		},
		{
			desc:  `only comments`,
			input: "# comment;\n/* comment */--comment\n/* comment */",
			want: nil,
		},
		{
			desc:  `second query ends in the middle of string`,
			input: `SELECT "123"; SELECT "45`,
			want: []InputStatement{
				{
					Statement: `SELECT "123"`,
					Delim:     DelimiterHorizontal,
				},
				{
					Statement: `SELECT "45`,
					Delim:     DelimiterUndefined,
				},
			},
		},
		{
			desc:  `totally incorrect query`,
			input: `a"""""""""'''''''''b`,
			want: []InputStatement{
				{
					Statement: `a"""""""""'''''''''b`,
					Delim:     DelimiterUndefined,
				},
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got := SeparateInput(tt.input)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(InputStatement{})); diff != "" {
				t.Errorf("difference in statements: (-want +got):\n%s", diff)
			}
		})
	}
}
