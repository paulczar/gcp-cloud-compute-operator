package controller

import (
	"github.com/paulczar/gcp-cloud-compute-operator/pkg/controller/address"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, address.Add)
}
