package protocol

import (
	"encoding/json"
	"fmt"
)

// Request - JSON-RPC request packet
type Request struct {
	Protocol string            `json:"jsonrpc"`
	ID       string            `json:"id,omitempty"`
	Method   string            `json:"method"`
	Params   map[string]string `json:"params"`
}

// JSON - convert struct to json
func (r *Request) JSON() (string, error) {
	r.Protocol = "2.0"
	bin, err := json.Marshal(r)
	return string(bin), err
}

// FromJSON - convert json to struct
func (r *Request) FromJSON(jsonString string) error {
	jsonBytes := []byte(jsonString)
	return json.Unmarshal(jsonBytes, r)
}

// String representation
func (r *Request) String() string {
	return fmt.Sprintf("id=%s method=%s params=%s", r.ID, r.Method, r.Params)
}

// Response - JSON-RPC response packet
type Response struct {
	Protocol string            `json:"jsonrpc"`
	ID       string            `json:"id"`
	Result   map[string]string `json:"result,omitempty"`
	Error    map[string]string `json:"error,omitempty"`
}

// JSON - convert struct to json
func (r *Response) JSON() (string, error) {
	r.Protocol = "2.0"
	bin, err := json.Marshal(r)
	return string(bin), err
}

// FromJSON - convert json to struct
func (r *Response) FromJSON(jsonString string) error {
	jsonBytes := []byte(jsonString)
	return json.Unmarshal(jsonBytes, r)
}

// String representation
func (r *Response) String() string {
	return fmt.Sprintf("id=%s result=%s error=%s", r.ID, r.Result, r.Error)
}
