package remote

import "encoding/json"

type JSONCodec struct{}

func (JSONCodec) Marshal(v interface{}) ([]byte, error)      { return json.Marshal(v) }
func (JSONCodec) Unmarshal(data []byte, v interface{}) error { return json.Unmarshal(data, v) }
func (JSONCodec) ContentType() string                        { return "application/json" }
