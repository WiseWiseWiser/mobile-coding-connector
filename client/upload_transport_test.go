package client

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWrapFlakyChunkTransport_retriesThenSucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tracker := &ChunkTransportTracker{Attempts: make(map[int]int)}
	httpClient := srv.Client()
	httpClient.Transport = WrapFlakyChunkTransport(httpClient.Transport, FlakyChunkTransportConfig{
		FailChunkIndex: 2,
		FailCount:      2,
		Tracker:        tracker,
	})

	for i := 0; i < 3; i++ {
		body, ctype := chunkMultipartBody(t, 2, []byte("data"))
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/files/upload/chunk", body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", ctype)
		_, err = httpClient.Do(req)
		if i < 2 {
			if err == nil {
				t.Fatalf("attempt %d: want transport error", i+1)
			}
			continue
		}
		if err != nil {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
	}
	if tracker.Attempts[2] != 3 {
		t.Fatalf("tracker attempts = %d, want 3", tracker.Attempts[2])
	}
}

func chunkMultipartBody(t *testing.T, idx int, chunk []byte) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("upload_id", "u1")
	_ = w.WriteField("chunk_index", fmt.Sprintf("%d", idx))
	part, err := w.CreateFormFile("chunk", fmt.Sprintf("chunk_%d", idx))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(chunk); err != nil {
		t.Fatal(err)
	}
	ctype := w.FormDataContentType()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes()), ctype
}