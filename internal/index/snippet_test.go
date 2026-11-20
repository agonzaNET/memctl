package index_test

import (
	"strings"
	"testing"

	"github.com/agonzaNET/memctl/internal/index"
)

// ä repeats a two-byte rune n times, so every case also proves that the
// window counts runes, not bytes.
func ä(n int) string { return strings.Repeat("ä", n) }

func TestSnippet_SPEC0002_R27_window(t *testing.T) {
	tests := []struct {
		name, query, title, line, want string
	}{
		{
			name:  "200 runes whole",
			query: "needle",
			line:  ä(99) + " needle " + ä(93),
			want:  ä(99) + " needle " + ä(93),
		},
		{
			// The cut at rune 200 splits the last word, which is dropped.
			name:  "match near the start",
			query: "needle",
			line:  ä(19) + " needle " + ä(473),
			want:  ä(19) + " needle…",
		},
		{
			name:  "match in the middle, partial words dropped",
			query: "needle",
			line:  ä(249) + " needle " + ä(243),
			want:  "…needle…",
		},
		{
			name:  "match near the end",
			query: "needle",
			line:  ä(479) + " needle " + ä(13),
			want:  "…needle " + ä(13),
		},
		{
			name:  "spaces at cut edges trimmed",
			query: "needle",
			line:  ä(153) + " " + ä(95) + " needle " + ä(95) + " " + ä(147),
			want:  "…" + ä(95) + " needle " + ä(95) + "…",
		},
		{
			name:  "compound query term",
			query: "IT-104",
			line:  ä(249) + " IT-104 " + ä(243),
			want:  "…IT-104…",
		},
		{
			// "104" starts at rune 253; the word around it is kept whole.
			name:  "part inside a compound",
			query: "104",
			line:  ä(249) + " IT-104 " + ä(243),
			want:  "…IT-104…",
		},
		{
			// "pa" and "pa-850" both start at rune 250. Centred on
			// "pa-850" the window is [153, 353) and both edges split a
			// word; centred on "pa" it would be [151, 351), whose edges
			// fall next to spaces and keep ä(97) and ä(94).
			name:  "earliest then longest token",
			query: "pa PA-850",
			line:  ä(151) + " " + ä(97) + " PA-850 " + ä(94) + " " + ä(148),
			want:  "…PA-850 " + ä(94) + "…",
		},
		{
			name:  "first of two matched terms",
			query: "alpha needle",
			line:  ä(99) + " needle " + ä(193) + " alpha " + ä(193),
			want:  "…needle…",
		},
		{
			// Dropping the partial word would empty the window.
			name:  "title-only hit cut at the end",
			query: "needle",
			title: "needle",
			line:  ä(300),
			want:  ä(200) + "…",
		},
		{
			// The token ends a word that starts at rune 251; moving the
			// left edge (299) to the next space (403) would drop it.
			name:  "cut that would drop the token stays",
			query: "IT-104",
			line:  ä(250) + " " + ä(144) + "(IT-104) " + ä(96),
			want:  "…" + ä(96) + "(IT-104)…",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := tt.title
			if title == "" {
				title = "t"
			}
			x := index.New()
			put(x, "n.md", "---\ntitle: "+title+"\n---\n"+tt.line+"\n")
			hits, err := hitsOf(x.Search(index.Query{Text: tt.query}))
			if err != nil || len(hits) != 1 {
				t.Fatalf("hits %v, err %v", hits, err)
			}
			if got := hits[0].Snippet; got != tt.want {
				t.Errorf("snippet\n got %q (%d runes)\nwant %q (%d runes)", got, len([]rune(got)), tt.want, len([]rune(tt.want)))
			}
			if hits[0].SnippetLine != 4 {
				t.Errorf("line %d, want 4", hits[0].SnippetLine)
			}
		})
	}
}
