// Package git simply anonymously imports all the various git implementations so the can be used.
package git

import (
	// Register Gitlab implementation
	_ "github.com/projectsyn/lieutenant-operator/pkg/git/gitlab"
)
