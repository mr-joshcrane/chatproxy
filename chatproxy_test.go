package chatproxy_test

import (
	"bytes"
	"flag"
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
	tc := testClient(t,
		chatproxy.WithFixedResponse(want),
		chatproxy.WithOutput(buf, io.Discard),
		chatproxy.WithTranscript(io.Discard),
	)
	chatproxy.NewChatGPTClient = func(...chatproxy.ClientOption) (*chatproxy.ChatGPTClient, error) { return tc, nil }
	chatproxy.Ask([]string{"What", "is", "the", "purpose", "of", "this", "test?"})
	got := buf.String()
	if !strings.Contains(got, want) {
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
	if !strings.Contains(got, want) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestChat(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	input := strings.NewReader("You help me test my Chat CLI\nRequest\nexit\n")
	response := "Fixed response"
	tc := testClient(t, chatproxy.WithFixedResponse(response), chatproxy.WithInput(input), chatproxy.WithTranscript(buf))
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
	if !strings.Contains(got, want) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestCommit(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	input := "Testing commit CLI"
	tc := testClient(t, chatproxy.WithFixedResponse(input), chatproxy.WithTranscript(buf))
	chatproxy.NewChatGPTClient = func(...chatproxy.ClientOption) (*chatproxy.ChatGPTClient, error) { return tc, nil }
	chatproxy.Commit()
	got := buf.String()
	want := "SYSTEM) PURPOSE: Please read the git diff provided and write an appropriate commit message.\n\tFocus on the lines that start with a + (line added) or - (line removed)\n"
	if !strings.Contains(got, want) {
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

func TestIncorrectToken(t *testing.T) {
	t.Parallel()
	client := testClient(t)
	_, err := client.Ask("This is a test")
	if err == nil {
		t.Fatal(err)
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

func TestChat_FileOperations(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	outfile, in := setupFileOperations(t)
	input := strings.NewReader(in)
	client := testClient(t,
		chatproxy.WithFixedResponse("Fixed response"),
		chatproxy.WithInput(input),
		chatproxy.WithTranscript(buf),
	)
	client.Chat()
	got := buf.String()
	if !strings.Contains(got, "SYSTEM) PURPOSE: This is the purpose") {
		t.Fatalf("wanted purpose, got %s", got)
	}
	if !strings.Contains(got, "This is the first file") {
		t.Fatalf("wanted first file, got %s", got)
	}
	if !strings.Contains(got, "This is the second file") {
		t.Fatalf("wanted second file, got %s", got)
	}
	exists, _ := os.Stat(outfile)
	if exists == nil {
		t.Fatalf("wanted %s to be created", outfile)
	}
	contents, err := os.ReadFile(outfile)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "Fixed response\n" {
		t.Fatalf("wanted %s, got %s", "Fixed response\n", string(contents))
	}

}

func TestTranscript(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	input := strings.NewReader("Return fixed responses\nQuestion?\nOther question?\nexit\n")
	client := testClient(t, chatproxy.WithTranscript(buf), chatproxy.WithInput(input), chatproxy.WithFixedResponse("Fixed response"))
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

var runIntegration = flag.Bool("integration", false, "if true, run integration tests")

func TestIntegration_StreamingResponse(t *testing.T) {
	t.Parallel()
	if !*runIntegration {
		t.Skip("skipping test; only run with -integration")
	}
	buf := new(bytes.Buffer)
	client, err := chatproxy.DefaultGPTClient(
		chatproxy.WithStreaming(true),
		chatproxy.WithOutput(buf, io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := client.Ask("This is a test")
	if err != nil {
		t.Fatal(err)
	}
	if len(answer) == 0 {
		t.Fatal("answer is empty")
	}
	if buf.Len() == 0 {
		t.Fatal("streaming response should stream the result to the output, did not.")
	}
}

func TestIntegration_BufferedResponse(t *testing.T) {
	t.Parallel()
	if !*runIntegration {
		t.Skip("skipping test; only run with -integration")
	}
	client, err := chatproxy.DefaultGPTClient(
		chatproxy.WithStreaming(false),
		chatproxy.WithOutput(io.Discard, io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := client.Ask("This is a test")
	if err != nil {
		t.Fatal(err)
	}
	if len(answer) == 0 {
		t.Fatal("answer is empty")
	}
}

var SuppressOutput = chatproxy.WithOutput(io.Discard, io.Discard)
var TestToken = chatproxy.WithToken("test-token")

func testConstructor(opts ...chatproxy.ClientOption) (*chatproxy.ChatGPTClient, error) {
	opts = append([]chatproxy.ClientOption{SuppressOutput, TestToken}, opts...)
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

func setupFileOperations(t *testing.T) (outfile, input string) {
	tempDir := t.TempDir()
	outfile = tempDir + "/outfile.txt"
	input = fmt.Sprintf("This is the purpose\n>%s\n<%s write a fixed response\nexit\n", tempDir, outfile)
	err := os.WriteFile(tempDir+"/file1.txt", []byte("This is the first file"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(tempDir+"/file2.txt", []byte("This is the second file"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	return outfile, input
}
