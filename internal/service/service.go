package service

import (
	"github.com/samber/do/v2"
)

var Package = do.Package(
	do.Lazy(runnerProvider),
	do.Lazy(restConfigProvider),
	do.Lazy(kubernetesProvider),
)
