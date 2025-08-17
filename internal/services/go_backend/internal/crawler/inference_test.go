package crawler

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockTransport struct {
	response *http.Response
	err      error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

func newTestClient(respBody string) *http.Client {
	rt := &mockTransport{response: &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(respBody)),
	}}
	return &http.Client{Transport: rt}
}

func TestDataQueryReturnsRawString(t *testing.T) {
	body := `{"choices":[{"message":{"content":"test response"}}]}`
	m := Manager{httpClient: newTestClient(body), LlmApiKey: "test"}
	out, err := m.DataQuery("prompt", "question", "data", nil)
	if err != nil {
		t.Fatalf("DataQuery returned error: %v", err)
	}
	if out != "test response" {
		t.Fatalf("expected 'test response', got %v", out)
	}
}

type sampleStruct struct {
	Msg string `json:"msg"`
}

func TestDataQueryWithStructure(t *testing.T) {
	body := `{"choices":[{"message":{"content":"{\\"msg\\":\\"hello\\"}"}}]}`
	m := Manager{httpClient: newTestClient(body)}
	out, err := m.DataQuery("You are parsing html.", "question: what's the text in this html", "data", sampleStruct{})
	if err != nil {
		t.Fatalf("DataQuery returned error: %v", err)
	}
	res, ok := out.(sampleStruct)
	if !ok {
		t.Fatalf("expected sampleStruct, got %T", out)
	}
	if res.Msg != "hello" {
		t.Fatalf("expected Msg 'hello', got %v", res.Msg)
	}
}

func TestDataQueryNoClient(t *testing.T) {
	m := Manager{}
	if _, err := m.DataQuery("p", "q", "d", nil); err == nil {
		t.Fatalf("expected error when httpClient is nil")
	}
}

type structExample struct {
	Name string
	Age  int
}

func TestStructureOutput(t *testing.T) {
	out := StrucutreOutput(structExample{})
	if !strings.Contains(out, "Name") || !strings.Contains(out, "Age") {
		t.Errorf("output should list field names, got: %s", out)
	}
	if !strings.Contains(out, "string") || !strings.Contains(out, "int") {
		t.Errorf("output should list field types, got: %s", out)
	}
}
