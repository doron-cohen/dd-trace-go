// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package datastreams

import (
	"compress/gzip"
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tinylib/msgp/msgp"
)

type fakeTransport struct {
	requests []*http.Request
}

func (t *fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, r)
	return &http.Response{StatusCode: 200}, nil
}

func toHeaders(headers map[string]string) http.Header {
	h := make(http.Header)
	for k, v := range headers {
		h[k] = []string{v}
	}
	return h
}

func TestHTTPTransport(t *testing.T) {
	p := StatsPayload{Env: "env-1", Stats: []StatsBucket{{
		Start:    2,
		Duration: 10,
		Stats: []StatsPoint{{
			Service:        "service-1",
			EdgeTags:       []string{"edge-1"},
			Hash:           1,
			ParentHash:     2,
			PathwayLatency: []byte{1, 2, 3},
			EdgeLatency:    []byte{4, 5, 6},
		}},
	}}}

	t.Run("agentless", func(t *testing.T) {
		fakeTransport := fakeTransport{}
		transport := newHTTPTransport("agent-address", "datadoghq.com", "key", &http.Client{Transport: &fakeTransport}, true)
		assert.Nil(t, transport.sendPipelineStats(&p))
		assert.Len(t, fakeTransport.requests, 1)
		r := fakeTransport.requests[0]
		assert.Equal(t, "https://trace.agent.datadoghq.com/api/v0.1/pipeline_stats", r.URL.String())
		goVersion := strings.TrimPrefix(runtime.Version(), "go")
		headers := toHeaders(map[string]string{
			"Content-Encoding":              "gzip",
			"Content-Type":                  "application/msgpack",
			"Datadog-Meta-Lang-Version":     goVersion,
			"Datadog-Meta-Lang":             "go",
			"Datadog-Meta-Lang-Interpreter": runtime.Compiler + "-" + runtime.GOARCH + "-" + runtime.GOOS,
			"Dd-Api-Key":                    "key",
		})
		assert.Equal(t, headers, r.Header)
		gzReader, err := gzip.NewReader(r.Body)
		assert.Nil(t, err)
		var sentPayload StatsPayload
		assert.Nil(t, msgp.Decode(gzReader, &sentPayload))
		assert.Equal(t, p, sentPayload)
	})

	t.Run("with_agent", func(t *testing.T) {
		fakeTransport := fakeTransport{}
		transport := newHTTPTransport("agent-address:8126", "datadoghq.com", "key", &http.Client{Transport: &fakeTransport}, false)
		assert.Nil(t, transport.sendPipelineStats(&p))
		assert.Len(t, fakeTransport.requests, 1)
		r := fakeTransport.requests[0]
		assert.Equal(t, "http://agent-address:8126/v0.1/pipeline_stats", r.URL.String())
	})
}
