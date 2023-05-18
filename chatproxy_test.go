package chatproxy_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/chatproxy"
)

var token string = os.Getenv("OPENAPI_TOKEN")

func TestReadFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/config.json"
	contents := `{
    config: "yes",
}`

	err := ioutil.WriteFile(path, []byte(contents), 0644)
	if err != nil {
		t.Fatal(err)
	}
	got, err := chatproxy.MessageFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := chatproxy.ChatMessage{
		Content: fmt.Sprintf("--%s--\n%s", path, contents),
		Role:    chatproxy.RoleUser,
	}
	if want != got {
		cmp.Diff(want, got)
	}
}

func TestReadDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	c1path := dir + "/config1.json"
	c1contents := `true`

	err := ioutil.WriteFile(c1path, []byte(c1contents), 0644)
	if err != nil {
		t.Fatal(err)
	}
	c2path := dir + "/config2.json"
	c2contents := `false`

	err = ioutil.WriteFile(c2path, []byte(c2contents), 0644)
	if err != nil {
		t.Fatal(err)
	}
	got, err := chatproxy.MessagesFromFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []chatproxy.ChatMessage{
		{
			Content: fmt.Sprintf("--%s--\n%s", c1path, c1contents),
			Role:    chatproxy.RoleUser,
		},
		{
			Content: fmt.Sprintf("--%s--\n%s", c2path, c2contents),
			Role:    chatproxy.RoleUser,
		},
	}
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestWriteFile(t *testing.T) {
	t.Parallel()
	path := t.TempDir() + "/temp.txt"
	messages := chatproxy.ChatMessage{
		Content: "This is some file output.",
		Role: chatproxy.RoleBot,
	}
	chatproxy.MessageToFile(messages, path)
	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(output)
	want := "This is some file output.\n"
	if want != got {
		t.Fatalf("wanted %s, got %s", want, got)
	}
}
