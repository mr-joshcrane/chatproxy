package chatproxy_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/chatproxy"
)

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
	want := fmt.Sprintf("--%s--\n%s", path, contents)
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
	got, err := chatproxy.MessageFromFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := fmt.Sprintf("--%s--\n%s\n--%s--\n%s\n", c1path, c1contents, c2path, c2contents)

	want := chatproxy.ChatMessage{
		Content: c,
		Role:    chatproxy.RoleUser,
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
		Role:    chatproxy.RoleBot,
	}
	err := chatproxy.MessageToFile(messages, path)
	if err != nil {
		t.Fatal(err)
	}
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

func TestRollBackMessage_HandlesZeroLengthContexts(t *testing.T) {
	t.Parallel()
	client, err := chatproxy.NewChatGPTClient("")
	if err != nil {
		t.Fatal(err)
	}
	client.RollbackLastMessage()
}

func TestRollBackMessage_HandlesMultiMessageContexts(t *testing.T) {
	t.Parallel()
	client, err := chatproxy.NewChatGPTClient("")
	if err != nil {
		t.Fatal(err)
	}
	client.SetPurpose("This is the purpose")
	client.RecordMessage(chatproxy.ChatMessage{
		Content: "This is the content",
		Role:    chatproxy.RoleUser,
	})
	messages := client.RollbackLastMessage()
	got := messages[len(messages)-1].Content
	want := "This is the purpose"
	if want != got {
		t.Fatalf("wanted %s, got %s", want, got)
	}
}
