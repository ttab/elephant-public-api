package liveconnect_test

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/ttab/elephant-public-api/live"
	"github.com/ttab/elephant-public-api/live/liveconnect"
)

// The clients are expressed in the plain service interface, which is what
// makes them interchangeable with an implementation of it. A regeneration
// that renames or drops an adapter fails to compile here.
var (
	_ live.Blogs         = liveconnect.NewBlogsServiceClient(nil, "")
	_ live.Configuration = liveconnect.NewConfigurationServiceClient(nil, "")
	_ live.Posts         = liveconnect.NewPostsServiceClient(nil, "")
)

// The handler adapters take the plain implementation and return the mount
// path together with the handler, which is the pair the API server registers.
var (
	_ func(live.Blogs, ...connect.HandlerOption) (string, http.Handler)         = liveconnect.NewBlogsServiceHandler
	_ func(live.Configuration, ...connect.HandlerOption) (string, http.Handler) = liveconnect.NewConfigurationServiceHandler
	_ func(live.Posts, ...connect.HandlerOption) (string, http.Handler)         = liveconnect.NewPostsServiceHandler
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
			Service: liveconnect.BlogsName,
			Handler: func() (string, http.Handler) {
				return liveconnect.NewBlogsServiceHandler(nil)
			},
		},
		{
			Service: liveconnect.ConfigurationName,
			Handler: func() (string, http.Handler) {
				return liveconnect.NewConfigurationServiceHandler(nil)
			},
		},
		{
			Service: liveconnect.PostsName,
			Handler: func() (string, http.Handler) {
				return liveconnect.NewPostsServiceHandler(nil)
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
