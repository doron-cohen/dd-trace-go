package memcache

import (
	"context"
	"net"
	"testing"
	"time"

	dnstrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/miekg/dns"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Integration struct {
	msg      *dns.Msg
	mux      *dns.ServeMux
	addr     string
	numSpans int
}

func New() *Integration {
	return &Integration{}
}

func (i *Integration) Name() string {
	return "contrib/miekg/dns"
}

func (i *Integration) Init(t *testing.T) func() {
	t.Helper()
	i.addr = getFreeAddr(t).String()
	server := &dns.Server{
		Addr:    i.addr,
		Net:     "udp",
		Handler: dnstrace.WrapHandler(&handler{t: t, ig: i}),
	}
	// start the traced server
	go func() {
		require.NoError(t, server.ListenAndServe())
	}()
	// wait for the server to be ready
	waitServerReady(t, server.Addr)
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		assert.NoError(t, server.ShutdownContext(ctx))
	}
	return cleanup
}

func (i *Integration) GenSpans(t *testing.T) {
	t.Helper()
	msg := newMessage()
	_, err := dnstrace.Exchange(msg, i.addr)
	require.NoError(t, err)
	i.numSpans++
}

func (i *Integration) NumSpans() int {
	return i.numSpans
}

func newMessage() *dns.Msg {
	m := new(dns.Msg)
	m.SetQuestion("miek.nl.", dns.TypeMX)
	return m
}

type handler struct {
	t  *testing.T
	ig *Integration
}

func (h *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	assert.NoError(h.t, w.WriteMsg(m))
	h.ig.numSpans++
}

func getFreeAddr(t *testing.T) net.Addr {
	li, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := li.Addr()
	require.NoError(t, li.Close())
	return addr
}

func waitServerReady(t *testing.T, addr string) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeoutChan := time.After(5 * time.Second)
	for {
		m := new(dns.Msg)
		m.SetQuestion("miek.nl.", dns.TypeMX)
		_, err := dns.Exchange(m, addr)
		if err == nil {
			break
		}

		select {
		case <-ticker.C:
			continue

		case <-timeoutChan:
			t.Fatal("timeout waiting for DNS server to be ready")
		}
	}
}