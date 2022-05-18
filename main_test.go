package main

import "testing"

func TestMapPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"gopkg.in/go-yaml/yaml", "github.com/go-yaml/yaml"},
		{"gopkg.in/go-yaml/yaml.v2", "github.com/go-yaml/yaml/v2"},
		{"gopkg.in/yaml", "github.com/go-yaml/yaml"},
		{"gopkg.in/yaml.v2", "github.com/go-yaml/yaml/v2"},
		{"golang.org/x/tools", "github.com/golang/tools"},
		{"golang.org/x/crypto", "github.com/golang/crypto"},
		{"google.golang.org/grpc", "github.com/grpc/grpc-go"},
		{"google.golang.org/protobuf", "github.com/protocolbuffers/protobuf-go"},
		{"google.golang.org/api", "github.com/googleapis/google-api-go-client"},
		{"google.golang.org/genproto", "github.com/googleapis/go-genproto"},
		{"go.opentelemetry.io/proto/otlp", "github.com/open-telemetry/opentelemetry-proto-go"},
		{"google.golang.org/appengine", "github.com/golang/appengine"},
		{"honnef.co/go/tools", "github.com/dominikh/go-tools"},
		{"honnef.co/go/tools/v2", "github.com/dominikh/go-tools/v2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			actual := MapPath(tt.input)
			if tt.expected != actual {
				t.Errorf("expected %s, got %s", tt.expected, actual)
			}
		})
	}
}

//
//func TestList(t *testing.T) {
//	router := mux.NewRouter()
//	router.HandleFunc("/{module:.+}/@v/list", ListHandler)
//
//	var tests = []struct {
//		path    string
//		code    int
//		content string
//	}{
//		{"github.com/wandel", 410, ""},
//		{"github.com/wandel/modprox_test", 200, "v0.1.0\nv0.2.0\nv1.0\nv1.0.0\nv1.0.1"},
//		{"github.com/wandel/modprox_test/v2", 200, "v2.0.0"},
//		{"github.com/wandel/modprox_test/v3", 200, "not found: module github.com/wandel/modprox_test/v3: no matching versions for query \"latest\""},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.path, func(t *testing.T) {
//			url := fmt.Sprintf("http://localhost:8000/%s/@v/list", tt.path)
//			r := httptest.NewRequest("GET", url, nil)
//			w := httptest.NewRecorder()
//			router.ServeHTTP(w, r)
//			if w.Code != tt.code {
//				t.Errorf("expected status code %d, got %d", tt.code, w.Code)
//				return
//			}
//
//			body, err := ioutil.ReadAll(w.Body)
//			if err != nil {
//				t.Fatalf("failed to read body: %+v\n", err)
//			}
//
//			actual := strings.Split(string(body), "\n")
//			expected := strings.Split(tt.content, "\n")
//
//			for _, x := range expected {
//				if !contains(x, actual) {
//					t.Errorf("expected version '%s' to be present, but was missing", x)
//				}
//			}
//
//			for _, x := range actual {
//				if !contains(x, expected) {
//					t.Errorf("unexpected version '%s' was present", x)
//				}
//			}
//		})
//	}
//}
//
//func TestMod(t *testing.T) {
//	t.Error("Not Implemented Yet")
//}
//
//func TestInfo(t *testing.T) {
//	t.Error("Not Implemented Yet")
//}
//
//func TestArchive(t *testing.T) {
//	t.Error("Not Implemented Yet")
//}
//
//func TestLatest(t *testing.T) {
//	t.Error("Not Implemented Yet")
//}
//
//func contains(e string, xs []string) bool {
//	for _, x := range xs {
//		log.Println(e, xs)
//		if e == x {
//			return true
//		}
//	}
//	return false
//}
