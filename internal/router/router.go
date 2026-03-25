package router

import (
	"fmt"
	"strings"

	"github.com/llm-proxy/internal/backend"
	"github.com/llm-proxy/internal/config"
)

type Router struct {
	routes         []config.RouteConfig
	defaultRoute   string
	backendFactory *backend.Factory
}

func New(routes []config.RouteConfig, defaultRoute string, factory *backend.Factory) *Router {
	return &Router{
		routes:         routes,
		defaultRoute:   defaultRoute,
		backendFactory: factory,
	}
}

func (r *Router) Resolve(model string) (backend.Backend, error) {
	for _, route := range r.routes {
		if matchModel(model, route.ModelPattern) {
			return r.backendFactory.Get(route.Backend)
		}
	}

	return r.backendFactory.Get(r.defaultRoute)
}

func matchModel(model, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if pattern == model {
		return true
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(model, prefix)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(model, suffix)
	}

	return false
}

func (r *Router) ListRoutes() []string {
	var routes []string
	for _, route := range r.routes {
		routes = append(routes, fmt.Sprintf("%s -> %s", route.ModelPattern, route.Backend))
	}
	return routes
}
