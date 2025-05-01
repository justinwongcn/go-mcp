package transport

import (
	"io"
	"testing"
)

func TestMockTransport(t *testing.T) {
	reader1, writer1 := io.Pipe()
	reader2, writer2 := io.Pipe()

	serverTransport := NewMockServerTransport(reader2, writer1)
	clientTransport := NewMockClientTransport(reader1, writer2)

	testTransport(t, clientTransport, serverTransport)
}
