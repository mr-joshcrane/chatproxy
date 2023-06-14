package chatproxy_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
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

	err := os.WriteFile(path, []byte(contents), 0644)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := chatproxy.MessageFromFile(path)
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
	client := testClient(t)
	dir := t.TempDir()
	c1path := dir + "/config1.json"
	c1contents := `true`

	err := os.WriteFile(c1path, []byte(c1contents), 0644)
	if err != nil {
		t.Fatal(err)
	}
	c2path := dir + "/config2.json"
	c2contents := `false`

	err = os.WriteFile(c2path, []byte(c2contents), 0644)
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.MessageFromFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("--%s--\n%s\n--%s--\n%s\n", c1path, c1contents, c2path, c2contents)

	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}

}

func TestWriteFile(t *testing.T) {
	t.Parallel()
	path := t.TempDir() + "/temp.txt"
	err := chatproxy.MessageToFile("This is some file output.", path)
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
	client := testClient(t)
	client.RollbackLastMessage()
}

func TestRollBackMessage_HandlesMultiMessageContexts(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	client.SetPurpose("This is the purpose")
	client.RecordMessage(chatproxy.RoleUser, "This is the content")
	messages := client.RollbackLastMessage()
	got := messages[len(messages)-1].Content
	want := "PURPOSE: This is the purpose"
	if want != got {
		t.Fatalf("wanted %s, got %s", want, got)
	}
}

func TestModeSwitch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		description string
		input       string
		want        chatproxy.Strategy
	}{
		{
			description: "User requests file load",
			input:       ">file.txt",
			want:        chatproxy.FileLoad{},
		},
		{
			description: "User requests file writen out",
			input:       "<file.txt and some random prompt",
			want:        chatproxy.FileWrite{},
		},
		{
			description: "Default case with no special action",
			input:       "How many brackets do I have <><><><><>",
			want:        chatproxy.Default{},
		},
		{
			description: "User requests comprehension questions",
			input:       "?",
			want:        chatproxy.Default{},
		},
	}
	client := testClient(t)
	for _, tc := range cases {
		got := client.GetStrategy(tc.input)
		if diff := cmp.Diff(got, tc.want, cmp.Transformer("TypeOnly", func(i chatproxy.Strategy) string {
			return fmt.Sprintf("%T", i)
		})); diff != "" {
			t.Errorf("(-want +got):\n%s", diff)
		}

	}

}

func TestTranscript(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	input := strings.NewReader("Return fixed responses\nQuestion?\nOther question?\nexit\n")
	client := testClient(t, chatproxy.WithAudit(buf), chatproxy.WithInput(input), chatproxy.WithFixedResponse("Fixed response"))
	client.Chat()
	want := []string{
		"SYSTEM) PURPOSE: Return fixed responses",
		"USER) Question?",
		"ASSISTANT) Fixed response",
		"USER) Other question?",
		"ASSISTANT) Fixed response",
		"USER) *exit*",
		"",
	}
	got := strings.Split(buf.String(), "\n")
	if !cmp.Equal(want, got) {
		t.Fatalf(cmp.Diff(want, got))
	}
}

var SuppressOutput = chatproxy.WithOutput(io.Discard, io.Discard)

func testClient(t *testing.T, opts ...chatproxy.ClientOption) *chatproxy.ChatGPTClient {
	opts = append(opts, SuppressOutput)
	client, err := chatproxy.NewChatGPTClient(opts...)
	if err != nil {
		t.Fatal(err)
	}
	return client

}
