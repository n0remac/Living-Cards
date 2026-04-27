package ollama

import (
	"reflect"
	"testing"
)

func TestParseModelList(t *testing.T) {
	t.Parallel()

	got := ParseModelList("NAME ID SIZE MODIFIED\nqwen2.5:3b abc 1GB now\nnomic-embed-text def 1GB now\nqwen2.5:3b ghi 1GB later\n")
	want := []ModelOption{{Name: "nomic-embed-text"}, {Name: "qwen2.5:3b"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseModelList() = %#v, want %#v", got, want)
	}
}
