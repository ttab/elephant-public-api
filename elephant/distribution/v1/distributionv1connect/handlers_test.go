package distributionv1connect_test

import (
	"net/http"
	"testing"

	"github.com/ttab/elephant-public-api/elephant/distribution/v1/distributionv1connect"
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
			Service: distributionv1connect.ArchiveServiceName,
			Handler: func() (string, http.Handler) {
				return distributionv1connect.NewArchiveServiceHandler(
					distributionv1connect.UnimplementedArchiveServiceHandler{})
			},
		},
		{
			Service: distributionv1connect.ConfigurationServiceName,
			Handler: func() (string, http.Handler) {
				return distributionv1connect.NewConfigurationServiceHandler(
					distributionv1connect.UnimplementedConfigurationServiceHandler{})
			},
		},
		{
			Service: distributionv1connect.ContentServiceName,
			Handler: func() (string, http.Handler) {
				return distributionv1connect.NewContentServiceHandler(
					distributionv1connect.UnimplementedContentServiceHandler{})
			},
		},
		{
			Service: distributionv1connect.DeliveryServiceName,
			Handler: func() (string, http.Handler) {
				return distributionv1connect.NewDeliveryServiceHandler(
					distributionv1connect.UnimplementedDeliveryServiceHandler{})
			},
		},
		{
			Service: distributionv1connect.SearchServiceName,
			Handler: func() (string, http.Handler) {
				return distributionv1connect.NewSearchServiceHandler(
					distributionv1connect.UnimplementedSearchServiceHandler{})
			},
		},
		{
			Service: distributionv1connect.SubscriptionsServiceName,
			Handler: func() (string, http.Handler) {
				return distributionv1connect.NewSubscriptionsServiceHandler(
					distributionv1connect.UnimplementedSubscriptionsServiceHandler{})
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
