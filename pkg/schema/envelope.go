// Package schema defines stable public JSON envelopes shared by CLI commands.
package schema

const Version = "1.0"

type Envelope struct {
	SchemaVersion string   `json:"schema_version"`
	Status        string   `json:"status"`
	Data          any      `json:"data,omitempty"`
	Error         *Problem `json:"error,omitempty"`
}

type Problem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewError(code, message string) Problem {
	return Problem{Code: code, Message: message}
}

func Success(data any) Envelope {
	return Envelope{SchemaVersion: Version, Status: "ok", Data: data}
}

func Failure(problem Problem) Envelope {
	return Envelope{SchemaVersion: Version, Status: "error", Error: &problem}
}
