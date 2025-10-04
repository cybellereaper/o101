package messagesorter

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleCapture = `<ServiceID TYPE="UBYT">123</ServiceID>
<ProtocolType TYPE="STR">ChatService</ProtocolType>
<_ProtocolInfo>
  <Metadata>stuff</Metadata>
</_ProtocolInfo>
<RECORD>
  <Ignored>This should be stripped</Ignored>
</RECORD>
<Message>Hello</Message>
<Message>World</Message>
<Message>Hello</Message>
<Message>AfterRecord</Message>
</Protocol>
`

func TestProcess(t *testing.T) {
	res, err := Process(sampleCapture)
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	if res.ServiceID != "123" {
		t.Fatalf("ServiceID mismatch: got %q", res.ServiceID)
	}
	if res.ServiceName != "ChatService" {
		t.Fatalf("ServiceName mismatch: got %q", res.ServiceName)
	}

	expected := []string{
		"Message>AfterRecord</Message",
		"Message>Hello</Message",
		"Message>World</Message",
	}
	if len(res.Messages) != len(expected) {
		t.Fatalf("expected %d messages, got %d", len(expected), len(res.Messages))
	}
	for i, msg := range expected {
		if res.Messages[i] != msg {
			t.Errorf("message %d mismatch: expected %q got %q", i, msg, res.Messages[i])
		}
	}
}

func TestProcessFileAndWriteNumbered(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.xml")
	if err := os.WriteFile(inputPath, []byte(sampleCapture), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	outputPath, res, err := ProcessFile(inputPath, "")
	if err != nil {
		t.Fatalf("ProcessFile returned error: %v", err)
	}

	expectedOutputPath := filepath.Join(dir, "123_ChatService.txt")
	if outputPath != expectedOutputPath {
		t.Fatalf("unexpected output path: got %q expected %q", outputPath, expectedOutputPath)
	}

	if res.ServiceID != "123" || res.ServiceName != "ChatService" {
		t.Fatalf("unexpected metadata from ProcessFile: %+v", res)
	}

	if len(res.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(res.Messages))
	}

	contents, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	expectedFile := "1: Message>AfterRecord</Message\n2: Message>Hello</Message\n3: Message>World</Message\n"
	if string(contents) != expectedFile {
		t.Fatalf("unexpected file contents:\n%s", contents)
	}
}

func TestProcessErrors(t *testing.T) {
	_, err := Process("<ProtocolType TYPE=\"STR\">Name</ProtocolType>")
	if err != errMissingServiceID {
		t.Fatalf("expected errMissingServiceID, got %v", err)
	}

	_, err = Process("<ServiceID TYPE=\"UBYT\">id</ServiceID>")
	if err != errMissingServiceName {
		t.Fatalf("expected errMissingServiceName, got %v", err)
	}

	_, err = Process("<ServiceID TYPE=\"UBYT\">id</ServiceID><ProtocolType TYPE=\"STR\">name</ProtocolType>")
	if err != errMissingProtocolInfo {
		t.Fatalf("expected errMissingProtocolInfo, got %v", err)
	}
}
