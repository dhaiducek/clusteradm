// Copyright Contributors to the Open Cluster Management project
package wait

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/url"
	"sync/atomic"
	"testing"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestIsFatalAPIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "not found", err: k8serrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "name"), want: false},
		{name: "forbidden", err: k8serrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "name", errors.New("no")), want: true},
		{name: "unauthorized", err: k8serrors.NewUnauthorized("no"), want: true},
		{name: "bad request", err: k8serrors.NewBadRequest("bad"), want: true},
		{name: "method not supported", err: k8serrors.NewMethodNotSupported(schema.GroupResource{Resource: "pods"}, "connect"), want: true},
		{name: "connection refused", err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}, want: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: false},
		{
			name: "x509 unknown authority",
			err:  &url.Error{Op: "Get", URL: "https://hub.example", Err: x509.UnknownAuthorityError{}},
			want: true,
		},
		{
			name: "x509 hostname",
			err:  x509.HostnameError{Host: "hub.example"},
			want: true,
		},
		{
			name: "tls certificate verification",
			err:  &tls.CertificateVerificationError{Err: errors.New("failed to verify certificate")},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsFatalAPIError(tt.err); got != tt.want {
				t.Fatalf("IsFatalAPIError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandlePollError(t *testing.T) {
	t.Parallel()

	retryable := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
	status := &atomic.Value{}
	status.Store("")
	done, err := HandlePollError(retryable, status)
	if done || err != nil {
		t.Fatalf("HandlePollError() retryable = (%v, %v), want (false, nil)", done, err)
	}
	if got := status.Load().(string); got != retryable.Error() {
		t.Fatalf("HandlePollError() status = %q, want %q", got, retryable.Error())
	}

	forbidden := k8serrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "name", errors.New("no"))
	done, err = HandlePollError(forbidden, status)
	if done || err == nil {
		t.Fatalf("HandlePollError() forbidden = (%v, %v), want (false, forbidden)", done, err)
	}

	done, err = HandlePollError(context.DeadlineExceeded, status)
	if done || err != nil {
		t.Fatalf("HandlePollError() deadline = (%v, %v), want (false, nil)", done, err)
	}

	done, err = HandlePollError(nil, status)
	if done || err != nil {
		t.Fatalf("HandlePollError() nil = (%v, %v), want (false, nil)", done, err)
	}
}

func TestTimeoutError(t *testing.T) {
	t.Parallel()

	if err := TimeoutError(nil, nil, "timed out waiting"); err != nil {
		t.Fatalf("TimeoutError(nil) = %v, want nil", err)
	}

	other := errors.New("boom")
	if err := TimeoutError(other, nil, "timed out waiting"); err != other {
		t.Fatalf("TimeoutError(other) = %v, want original error", err)
	}

	err := TimeoutError(context.DeadlineExceeded, nil, "timed out waiting")
	if err == nil || err.Error() != "timed out waiting" {
		t.Fatalf("TimeoutError(deadline, empty) = %v, want timed out waiting", err)
	}

	status := &atomic.Value{}
	status.Store("connection refused")
	err = TimeoutError(context.DeadlineExceeded, status, "timed out waiting for %s", "operator")
	want := "timed out waiting for operator (connection refused)"
	if err == nil || err.Error() != want {
		t.Fatalf("TimeoutError() = %v, want %s", err, want)
	}
}
