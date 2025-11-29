package useragent

import (
	"fmt"
	"runtime"

	"github.com/docker/cagent/pkg/version"
)

var Header = fmt.Sprintf("Cagent/%s (%s; %s)", version.Version, runtime.GOOS, runtime.GOARCH)
