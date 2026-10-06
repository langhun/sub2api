package service

import "github.com/gin-gonic/gin"

// OpenAIResponsesLastVisibleSequenceKey stores the highest Responses
// sequence_number that the gateway has relayed for the current request.
//
// The service layer owns the SSE forwarding loop, while the HTTP handler owns
// the final error fallback. Keeping this small piece of state on the Gin
// context lets the handler append a protocol-correct response.failed event
// after a clean upstream EOF without parsing bytes already written to the
// client.
const OpenAIResponsesLastVisibleSequenceKey = "openai_responses_last_visible_sequence"

// SetOpenAIResponsesLastVisibleSequenceNumber records a sequence number that
// has been (or is about to be) relayed to the downstream Responses stream.
// Sequence numbers are monotonic for a request; stale values from a retry
// attempt can never move the request-level value backwards.
func SetOpenAIResponsesLastVisibleSequenceNumber(c *gin.Context, sequenceNumber int64) {
	if c == nil || sequenceNumber < 0 {
		return
	}
	if current, ok := OpenAIResponsesLastVisibleSequenceNumber(c); ok && current >= sequenceNumber {
		return
	}
	c.Set(OpenAIResponsesLastVisibleSequenceKey, sequenceNumber)
}

// OpenAIResponsesLastVisibleSequenceNumber returns the highest sequence
// number recorded for the current request.
func OpenAIResponsesLastVisibleSequenceNumber(c *gin.Context) (int64, bool) {
	if c == nil {
		return 0, false
	}
	value, ok := c.Get(OpenAIResponsesLastVisibleSequenceKey)
	if !ok {
		return 0, false
	}
	sequenceNumber, ok := value.(int64)
	return sequenceNumber, ok && sequenceNumber >= 0
}

// OpenAIResponsesNextVisibleSequenceNumber returns the sequence number to use
// for a synthesized terminal event. Responses streams start at zero when no
// upstream sequence has been relayed.
func OpenAIResponsesNextVisibleSequenceNumber(c *gin.Context) int64 {
	last, ok := OpenAIResponsesLastVisibleSequenceNumber(c)
	if !ok {
		return 0
	}
	return last + 1
}
