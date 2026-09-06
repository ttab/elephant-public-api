package distributionconnect_test

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/ttab/elephant-public-api/distribution"
	"github.com/ttab/elephant-public-api/distribution/distributionconnect"
)

// The clients are expressed in the plain service interface, which is what
// makes them interchangeable with an implementation of it. A regeneration
// that renames or drops an adapter fails to compile here.
var (
	_ distribution.Archive       = distributionconnect.NewArchiveServiceClient(nil, "")
	_ distribution.Configuration = distributionconnect.NewConfigurationServiceClient(nil, "")
	_ distribution.Content       = distributionconnect.NewContentServiceClient(nil, "")
	_ distribution.Delivery      = distributionconnect.NewDeliveryServiceClient(nil, "")
	_ distribution.Search        = distributionconnect.NewSearchServiceClient(nil, "")
	_ distribution.Subscriptions = distributionconnect.NewSubscriptionsServiceClient(nil, "")
)

// The handler adapters take the plain implementation and return the mount
// path together with the handler, which is the pair the API server registers.
var (
	_ func(distribution.Archive, ...connect.HandlerOption) (string, http.Handler)       = distributionconnect.NewArchiveServiceHandler
	_ func(distribution.Configuration, ...connect.HandlerOption) (string, http.Handler) = distributionconnect.NewConfigurationServiceHandler
	_ func(distribution.Content, ...connect.HandlerOption) (string, http.Handler)       = distributionconnect.NewContentServiceHandler
	_ func(distribution.Delivery, ...connect.HandlerOption) (string, http.Handler)      = distributionconnect.NewDeliveryServiceHandler
	_ func(distribution.Search, ...connect.HandlerOption) (string, http.Handler)        = distributionconnect.NewSearchServiceHandler
	_ func(distribution.Subscriptions, ...connect.HandlerOption) (string, http.Handler) = distributionconnect.NewSubscriptionsServiceHandler
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
			Service: distributionconnect.ArchiveName,
			Handler: func() (string, http.Handler) {
				return distributionconnect.NewArchiveServiceHandler(nil)
			},
		},
		{
			Service: distributionconnect.ConfigurationName,
			Handler: func() (string, http.Handler) {
				return distributionconnect.NewConfigurationServiceHandler(nil)
			},
		},
		{
			Service: distributionconnect.ContentName,
			Handler: func() (string, http.Handler) {
				return distributionconnect.NewContentServiceHandler(nil)
			},
		},
		{
			Service: distributionconnect.DeliveryName,
			Handler: func() (string, http.Handler) {
				return distributionconnect.NewDeliveryServiceHandler(nil)
			},
		},
		{
			Service: distributionconnect.SearchName,
			Handler: func() (string, http.Handler) {
				return distributionconnect.NewSearchServiceHandler(nil)
			},
		},
		{
			Service: distributionconnect.SubscriptionsName,
			Handler: func() (string, http.Handler) {
				return distributionconnect.NewSubscriptionsServiceHandler(nil)
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
