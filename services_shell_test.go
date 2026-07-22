package cloud

import (
	"bytes"
	"net/url"
	"os"
	"testing"
)

func TestReaderIsTerminalRejectsPipedInput(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()

	if readerIsTerminal(reader) {
		t.Fatal("pipe must use non-TTY shell mode")
	}
	if readerIsTerminal(bytes.NewBufferString("echo ok\n")) {
		t.Fatal("non-file reader must use non-TTY shell mode")
	}
}

func TestShellWSURLIncludesTerminalMode(t *testing.T) {
	client := NewClient(WithBaseURL("https://api.example.test"), WithToken("secret"))
	services := client.Workspace("ws").Project("project").Env("env").Services()

	rawURL, err := services.shellWSURL("service", false)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Query().Get("tty"); got != "false" {
		t.Fatalf("tty query = %q, want false", got)
	}
}
