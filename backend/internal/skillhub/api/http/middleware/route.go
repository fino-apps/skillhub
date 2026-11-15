package middleware

import "net/http"

type RouteMiddleware func(Route) []Middleware

// Route describes a single HTTP binding at registration time.
type Route struct {
	MethodName  string
	BindingName string
	Method      string
	Path        string
}

// WrapRoute applies route-specific middleware without affecting the global chain.
func WrapRoute(route Route, next http.Handler) http.Handler {
	return Chain(next, RouteMiddlewares(route)...)
}

func RouteMiddlewares(route Route) []Middleware {
	return route.Use(bindingMiddleware)
}

func (route Route) Use(groups ...RouteMiddleware) []Middleware {
	middlewares := make([]Middleware, 0, len(groups))
	for _, group := range groups {
		if group != nil {
			middlewares = append(middlewares, group(route)...)
		}
	}

	return middlewares
}

func bindingMiddleware(route Route) []Middleware {
	switch route.BindingName {
	case "ListSkills":
		return listSkillsMiddleware(route)
	case "GetSkill":
		return getSkillMiddleware(route)
	case "UpdateSkill":
		return updateSkillMiddleware(route)
	case "DeleteSkill":
		return deleteSkillMiddleware(route)
	case "ListVersions":
		return listVersionsMiddleware(route)
	case "CreateVersion":
		return createVersionMiddleware(route)
	case "ScanProject":
		return scanProjectMiddleware(route)
	case "PreviewImport":
		return previewImportMiddleware(route)
	case "ImportSkill":
		return importSkillMiddleware(route)
	case "CompareSkills":
		return compareSkillsMiddleware(route)
	case "GetInstallCommand":
		return getInstallCommandMiddleware(route)
	default:
		return nil
	}
}

func listSkillsMiddleware(Route) []Middleware {
	return nil
}

func getSkillMiddleware(Route) []Middleware {
	return nil
}

func updateSkillMiddleware(Route) []Middleware {
	return nil
}

func deleteSkillMiddleware(Route) []Middleware {
	return nil
}

func listVersionsMiddleware(Route) []Middleware {
	return nil
}

func createVersionMiddleware(Route) []Middleware {
	return nil
}

func scanProjectMiddleware(Route) []Middleware {
	return nil
}

func previewImportMiddleware(Route) []Middleware {
	return nil
}

func importSkillMiddleware(Route) []Middleware {
	return nil
}

func compareSkillsMiddleware(Route) []Middleware {
	return nil
}

func getInstallCommandMiddleware(Route) []Middleware {
	return nil
}
