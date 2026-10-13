package hubv1connect_test

import (
	"net/http"
	"testing"

	"github.com/ttab/elephant-public-api/elephant/hub/v1/hubv1connect"
)

// TestHandlerPaths pins the Connect mount paths. They are the unprefixed
// /<package>.<Service>/ roots, which is what every Connect client and proxy
// assumes, and an ingress rule written against them is only correct for as
// long as this holds.
func TestHandlerPaths(t *testing.T) {
	cases := []struct {
		Service string
		Handler func() (string, http.Handler)
	}{
		{
			Service: hubv1connect.ClientServiceName,
			Handler: func() (string, http.Handler) {
				return hubv1connect.NewClientServiceHandler(
					hubv1connect.UnimplementedClientServiceHandler{})
			},
		},
		{
			Service: hubv1connect.HubServiceName,
			Handler: func() (string, http.Handler) {
				return hubv1connect.NewHubServiceHandler(
					hubv1connect.UnimplementedHubServiceHandler{})
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Service, func(t *testing.T) {
			path, handler := c.Handler()

			want := "/" + c.Service + "/"
			if path != want {
				t.Errorf("got the mount path %q, wanted %q", path, want)
			}

			if handler == nil {
				t.Error("got a nil handler")
			}
		})
	}
}
