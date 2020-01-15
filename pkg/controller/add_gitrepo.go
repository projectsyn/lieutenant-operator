package controller

import (
	"github.com/projectsyn/lieutenant-operator/pkg/controller/gitrepo"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, gitrepo.Add)
}
