package mobilewebrtc

import (
	"context"
	"testing"
)

func TestPeerCloseDoesNotCancelStreamRequests(t *testing.T) {
	peer := newTestPeerSession(t)
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	streamCtx, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()

	peer.trackRequest("request-1", cancelRequest)
	peer.trackStreamRequest("stream-1", cancelStream)
	peer.close()

	assertContextDone(t, requestCtx, "request context")
	assertContextOpen(t, streamCtx, "stream context")
}

func TestCancelRequestCancelsStreamRequest(t *testing.T) {
	peer := newTestPeerSession(t)
	streamCtx, cancelStream := context.WithCancel(context.Background())
	peer.trackStreamRequest("stream-1", cancelStream)

	peer.cancelRequest("stream-1")

	assertContextDone(t, streamCtx, "stream context")
}

func newTestPeerSession(t *testing.T) *peerSession {
	t.Helper()
	transportCtx, cancelTransport := context.WithCancel(context.Background())
	t.Cleanup(cancelTransport)
	peerCtx, cancelPeer := context.WithCancel(transportCtx)
	return &peerSession{
		transport: &Transport{
			ctx:   transportCtx,
			peers: make(map[string]*peerSession),
		},
		ctx:            peerCtx,
		cancel:         cancelPeer,
		requests:       make(map[string]context.CancelFunc),
		streamRequests: make(map[string]context.CancelFunc),
	}
}

func assertContextDone(t *testing.T, ctx context.Context, label string) {
	t.Helper()
	select {
	case <-ctx.Done():
	default:
		t.Fatalf("%s was not cancelled", label)
	}
}

func assertContextOpen(t *testing.T, ctx context.Context, label string) {
	t.Helper()
	select {
	case <-ctx.Done():
		t.Fatalf("%s was cancelled", label)
	default:
	}
}
