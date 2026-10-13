package assetsv1connect_test

import (
	"net/http"
	"testing"

	"github.com/ttab/elephant-public-api/elephant/assets/v1/assetsv1connect"
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
			Service: assetsv1connect.KeysServiceName,
			Handler: func() (string, http.Handler) {
				return assetsv1connect.NewKeysServiceHandler(
					assetsv1connect.UnimplementedKeysServiceHandler{})
			},
		},
		{
			Service: assetsv1connect.ManagementServiceName,
			Handler: func() (string, http.Handler) {
				return assetsv1connect.NewManagementServiceHandler(
					assetsv1connect.UnimplementedManagementServiceHandler{})
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
