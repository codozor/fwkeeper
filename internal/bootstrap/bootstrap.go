package bootstrap

import (
	"github.com/samber/do/v2"

	"github.com/codozor/fwkeeper/internal/logger"
)

var Package = do.Package(
	logger.Package,
	Providers,
)
