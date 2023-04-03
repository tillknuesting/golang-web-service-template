package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func BenchmarkSetHandler(b *testing.B) {
	kvStore := KeyValueStore{
		kvMap: make(map[Key]Value),
	}

	var payload SetRequest
	var req *http.Request

	// 1 byte value size
	payload = SetRequest{
		Key:   "benchmark-key-1byte",
		Value: Value(string(make([]byte, 1))), // 1 byte
	}
	body, _ := json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/set", bytes.NewBuffer(body))

	b.Run("1 byte", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			kvStore.SetHandler(w, req)
		}
	})

	// 1KB value size
	payload = SetRequest{
		Key:   "benchmark-key-1kb",
		Value: Value(string(make([]byte, 1024))), // 1KB
	}
	body, _ = json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/set", bytes.NewBuffer(body))

	b.Run("1KB", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			kvStore.SetHandler(w, req)
		}
	})

	// 100KB value size
	payload = SetRequest{
		Key:   "benchmark-key-100kb",
		Value: Value(string(make([]byte, 100*1024))), // 100KB
	}
	body, _ = json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/set", bytes.NewBuffer(body))

	b.Run("100KB", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			kvStore.SetHandler(w, req)
		}
	})

	// 1MB value size
	payload = SetRequest{
		Key:   "benchmark-key-1mb",
		Value: Value(string(make([]byte, 1024*1024))), // 1MB
	}
	body, _ = json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/set", bytes.NewBuffer(body))

	b.Run("1MB", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			kvStore.SetHandler(w, req)
		}
	})
}

func BenchmarkGetHandler(b *testing.B) {
	kvStore := KeyValueStore{
		kvMap: map[Key]Value{
			"benchmark-key": "benchmark-value",
		},
	}

	var payload GetRequest
	var req *http.Request

	// 1 byte value size
	kvStore.kvMap["benchmark-key"] = Value(string(make([]byte, 1)))

	payload = GetRequest{
		Key: "benchmark-key",
	}

	body, _ := json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/get", bytes.NewBuffer(body))

	b.Run("1Byte", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			kvStore.GetHandler(w, req)
		}
	})

	// 1KB value size
	kvStore.kvMap["benchmark-key"] = Value(string(make([]byte, 1024)))

	payload = GetRequest{
		Key: "benchmark-key",
	}
	body, _ = json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/get", bytes.NewBuffer(body))

	b.Run("1KB", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			kvStore.GetHandler(w, req)
		}
	})

	// 100KB value size
	kvStore.kvMap["benchmark-key"] = Value(string(make([]byte, 100*1024)))
	payload = GetRequest{
		Key: "benchmark-key",
	}
	body, _ = json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/get", bytes.NewBuffer(body))

	b.Run("100KB", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			kvStore.GetHandler(w, req)
		}
	})

	// 1MB value size
	kvStore.kvMap["benchmark-key"] = Value(string(make([]byte, 1024*1024)))
	payload = GetRequest{
		Key: "benchmark-key",
	}
	body, _ = json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/get", bytes.NewBuffer(body))

	b.Run("1MB", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			kvStore.GetHandler(w, req)
		}
	})
}

func TestUseEnvOrDefaultIfNotSet(t *testing.T) {
	// Test case 1: environment variable is set
	os.Setenv("ENV_VAR", "10s")
	defer os.Unsetenv("ENV_VAR")
	expected := 10 * time.Second
	duration, _ := time.ParseDuration(os.Getenv("ENV_VAR"))
	result := useEnvOrDefaultIfNotSet(duration, expected)
	if result != expected {
		t.Errorf("Test case 1 failed: useEnvOrDefaultIfNotSet() returned %v, expected %v", result, expected)
	}

	// Test case 2: environment variable is not set
	os.Unsetenv("ENV_VAR")
	expected = 20 * time.Second
	result = useEnvOrDefaultIfNotSet(os.Getenv("ENV_VAR"), expected)
	if result != expected {
		t.Errorf("Test case 2 failed: useEnvOrDefaultIfNotSet() returned %v, expected %v", result, expected)
	}

	// Test case 3: environment variable is empty string
	os.Setenv("ENV_VAR", "")
	defer os.Unsetenv("ENV_VAR")
	expected = 30 * time.Second
	result = useEnvOrDefaultIfNotSet(os.Getenv("ENV_VAR"), expected)
	if result != expected {
		t.Errorf("Test case 3 failed: useEnvOrDefaultIfNotSet() returned %v, expected %v", result, expected)
	}

	// Test case 4: unexpected type passed as envValue
	unexpected := []int{1, 2, 3}
	expectedError := "unexpected type []int"
	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("%v", r)
			if errStr != expectedError {
				t.Errorf("Test case 4 failed: useEnvOrDefaultIfNotSet() panicked with message '%v', expected '%v'", errStr, expectedError)
			}
		} else {
			t.Errorf("Test case 4 failed: useEnvOrDefaultIfNotSet() did not panic as expected")
		}
	}()
	useEnvOrDefaultIfNotSet(unexpected, expected)
}

func TestKeyValueStore_SetHandler(t *testing.T) {
	type fields struct {
		Mutex sync.Mutex
		kvMap map[Key]Value
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		expectedMap map[Key]Value
		expectedMsg string
		expectError bool
	}{
		{
			name: "valid request",
			fields: fields{
				Mutex: sync.Mutex{},
				kvMap: map[Key]Value{},
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/set", bytes.NewBufferString(`{"key":"test", "value":"value"}`)),
			},
			expectedMap: map[Key]Value{"test": "value"},
			expectedMsg: "202" + "\n",
			expectError: false,
		},
		{
			name: "invalid request body",
			fields: fields{
				Mutex: sync.Mutex{},
				kvMap: map[Key]Value{},
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/set", bytes.NewBufferString(`"value":"value"}`)),
			},
			expectedMap: map[Key]Value{},
			expectedMsg: "json: cannot unmarshal string into Go value of type main.SetRequest",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kv := &KeyValueStore{
				Mutex: tt.fields.Mutex,
				kvMap: tt.fields.kvMap,
			}

			kv.SetHandler(tt.args.w, tt.args.r)

			if tt.expectError {
				resp := tt.args.w.(*httptest.ResponseRecorder)
				if resp.Code != http.StatusBadRequest {
					t.Errorf("expected status %v but got %v", http.StatusBadRequest, resp.Code)
				}
				if !strings.Contains(resp.Body.String(), tt.expectedMsg) {
					t.Errorf("expected message %v but got %v", tt.expectedMsg, resp.Body.String())
				}
				return
			}

			if !reflect.DeepEqual(kv.kvMap, tt.expectedMap) {
				t.Errorf("expected map %v but got %v", tt.expectedMap, kv.kvMap)
			}

			resp := tt.args.w.(*httptest.ResponseRecorder)
			if resp.Code != http.StatusOK {
				t.Errorf("expected status %v but got %v", http.StatusOK, resp.Code)
			}
			if resp.Body.String() != tt.expectedMsg {
				t.Errorf("expected message %v but got %v", tt.expectedMsg, resp.Body.String())
			}
		})
	}
}
