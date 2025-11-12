package bootstrap

import (
	"github.com/samber/do/v2"

	"github.com/codozor/fwkeeper/internal/service"
	"github.com/codozor/fwkeeper/internal/logger"
)

var Package = do.Package(
	service.Package,
	logger.Package,
)
