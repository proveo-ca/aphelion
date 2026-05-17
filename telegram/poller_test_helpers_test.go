//go:build linux

package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func decodeJSONRequest(t *testing.T, req *http.Request, out any) {
	t.Helper()
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
}

type testPollerCheckpoint struct {
	next         int64
	states       map[int64]PollerUpdateState
	saved        []int64
	accepted     []PollerAccepted
	handled      []int64
	failures     []PollerFailure
	terminals    []PollerTerminal
	ops          []string
	acceptErr    error
	acceptResult PollerAcceptResult
}

func (c *testPollerCheckpoint) NextUpdateID(context.Context) (int64, error) {
	return c.next, nil
}

func (c *testPollerCheckpoint) SaveNextUpdateID(_ context.Context, nextUpdateID int64) error {
	c.ops = append(c.ops, "save")
	c.saved = append(c.saved, nextUpdateID)
	return nil
}

func (c *testPollerCheckpoint) UpdateState(_ context.Context, updateID int64) (PollerUpdateState, error) {
	if c.states == nil {
		return PollerUpdateState{}, nil
	}
	state := c.states[updateID]
	if state.Found || state.Terminal || state.Status != "" {
		c.ops = append(c.ops, "state")
	}
	return state, nil
}

func (c *testPollerCheckpoint) RecordAccepted(_ context.Context, accepted PollerAccepted) (PollerAcceptResult, error) {
	if c.acceptErr != nil {
		return PollerAcceptResult{}, c.acceptErr
	}
	c.ops = append(c.ops, "accepted")
	c.accepted = append(c.accepted, accepted)
	if c.acceptResult != (PollerAcceptResult{}) {
		return c.acceptResult, nil
	}
	return PollerAcceptResult{Dispatch: true}, nil
}

func (c *testPollerCheckpoint) RecordHandled(_ context.Context, updateID int64) error {
	c.ops = append(c.ops, "handled")
	c.handled = append(c.handled, updateID)
	return nil
}

func (c *testPollerCheckpoint) RecordFailure(_ context.Context, failure PollerFailure) error {
	c.ops = append(c.ops, "failure")
	c.failures = append(c.failures, failure)
	return nil
}

func (c *testPollerCheckpoint) RecordTerminal(_ context.Context, terminal PollerTerminal) error {
	c.ops = append(c.ops, "terminal")
	c.terminals = append(c.terminals, terminal)
	return nil
}
