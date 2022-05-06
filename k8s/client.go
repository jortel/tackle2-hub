package k8s

import (
	liberr "github.com/konveyor/controller/pkg/error"
	"k8s.io/client-go/kubernetes/scheme"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

//
// NewClient builds new k8s client.
func NewClient() (newClient k8s.Client, err error) {
	cfg, _ := config.GetConfig()
	newClient, err = k8s.New(
		cfg,
		k8s.Options{
			Scheme: scheme.Scheme,
		})
	if err != nil {
		err = liberr.Wrap(err)
	}
	return
}
