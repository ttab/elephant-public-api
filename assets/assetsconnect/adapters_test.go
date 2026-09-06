package assetsconnect_test

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/ttab/elephant-public-api/assets"
	"github.com/ttab/elephant-public-api/assets/assetsconnect"
)

// The clients are expressed in the plain service interface, which is what
// makes them interchangeable with an implementation of it. A regeneration
// that renames or drops an adapter fails to compile here.
var (
	_ assets.Keys       = assetsconnect.NewKeysServiceClient(nil, "")
	_ assets.Management = assetsconnect.NewManagementServiceClient(nil, "")
)

// The handler adapters take the plain implementation and return the mount
// path together with the handler, which is the pair the API server registers.
var (
	_ func(assets.Keys, ...connect.HandlerOption) (string, http.Handler)       = assetsconnect.NewKeysServiceHandler
	_ func(assets.Management, ...connect.HandlerOption) (string, http.Handler) = assetsconnect.NewManagementServiceHandler
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
			Service: assetsconnect.KeysName,
			Handler: func() (string, http.Handler) {
				return assetsconnect.NewKeysServiceHandler(nil)
			},
		},
		{
			Service: assetsconnect.ManagementName,
			Handler: func() (string, http.Handler) {
				return assetsconnect.NewManagementServiceHandler(nil)
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
