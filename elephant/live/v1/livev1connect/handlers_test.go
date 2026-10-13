package livev1connect_test

import (
	"net/http"
	"testing"

	"github.com/ttab/elephant-public-api/elephant/live/v1/livev1connect"
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
			Service: livev1connect.BlogsServiceName,
			Handler: func() (string, http.Handler) {
				return livev1connect.NewBlogsServiceHandler(
					livev1connect.UnimplementedBlogsServiceHandler{})
			},
		},
		{
			Service: livev1connect.PostsServiceName,
			Handler: func() (string, http.Handler) {
				return livev1connect.NewPostsServiceHandler(
					livev1connect.UnimplementedPostsServiceHandler{})
			},
		},
		{
			Service: livev1connect.ConfigurationServiceName,
			Handler: func() (string, http.Handler) {
				return livev1connect.NewConfigurationServiceHandler(
					livev1connect.UnimplementedConfigurationServiceHandler{})
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
