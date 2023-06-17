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

func TestAsk(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	want := "To test the Ask CLI"
	tc := testClient(t, chatproxy.WithFixedResponse(want), chatproxy.WithOutput(buf, io.Discard))
	chatproxy.NewChatGPTClient = func(...chatproxy.ClientOption) (*chatproxy.ChatGPTClient, error) { return tc, nil }
	chatproxy.Ask([]string{"What", "is", "the", "purpose", "of", "this", "test?"})
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestCard(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	want := "To test the Card CLI"
	tc := testClient(t, chatproxy.WithFixedResponse(want), chatproxy.WithOutput(buf, os.Stderr))
	chatproxy.NewChatGPTClient = func(...chatproxy.ClientOption) (*chatproxy.ChatGPTClient, error) { return tc, nil }
	chatproxy.Card([]string{"card", "www.example.com"})
	got := buf.String()
	want = fmt.Sprintf(`[%s]`, want)
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestChat(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	input := strings.NewReader("You help me test my Chat CLI\nRequest\nexit\n")
	response := "Fixed response"
	tc := testClient(t, chatproxy.WithFixedResponse(response), chatproxy.WithInput(input), chatproxy.WithAudit(buf))
	chatproxy.NewChatGPTClient = func(...chatproxy.ClientOption) (*chatproxy.ChatGPTClient, error) { return tc, nil }
	chatproxy.Chat()
	got := buf.String()
	want := "SYSTEM) PURPOSE: You help me test my Chat CLI\nUSER) Request\nASSISTANT) Fixed response\nUSER) *exit*\n"
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestTLDR(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	want := "To test the TLDR CLI"
	tc := testClient(t, chatproxy.WithFixedResponse(want), chatproxy.WithOutput(buf, io.Discard))
	chatproxy.NewChatGPTClient = func(...chatproxy.ClientOption) (*chatproxy.ChatGPTClient, error) { return tc, nil }
	chatproxy.TLDR([]string{"tldr", "www.example.com"})
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestCommit(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	input := "Testing commit CLI"
	tc := testClient(t, chatproxy.WithFixedResponse(input), chatproxy.WithAudit(buf))
	chatproxy.NewChatGPTClient = func(...chatproxy.ClientOption) (*chatproxy.ChatGPTClient, error) { return tc, nil }
	chatproxy.Commit()
	got := buf.String()
	want := "SYSTEM) PURPOSE: Please read the git diff provided and write an appropriate commit message.\n\tFocus on the lines that start with a + (line added) or - (line removed)\n"
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

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
	msg := client.RollbackLastMessage()
	if !cmp.Equal(msg, []chatproxy.ChatMessage{}) {
		t.Fatalf("wanted empty ChatMessageArray, got %v", msg)
	}
}

func TestRollBackMessage_HandlesSingleMessageContexts(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	client.SetPurpose("This is the purpose")
	messages := client.RollbackLastMessage()
	got := messages[len(messages)-1].Content
	want := "PURPOSE: This is the purpose"
	if want != got {
		t.Fatalf("wanted %s, got %s", want, got)
	}
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
			description: "User requests file written out",
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

func testConstructor(opts ...chatproxy.ClientOption) (*chatproxy.ChatGPTClient, error) {
	opts = append([]chatproxy.ClientOption{SuppressOutput}, opts...)
	return chatproxy.DefaultGPTClient(opts...)
}

func testClient(t *testing.T, opts ...chatproxy.ClientOption) *chatproxy.ChatGPTClient {
	chatproxy.NewChatGPTClient = testConstructor
	client, err := chatproxy.NewChatGPTClient(opts...)
	if err != nil {
		t.Fatal(err)
	}
	return client

}
