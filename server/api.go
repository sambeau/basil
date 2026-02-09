package server

import (
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
	"github.com/sambeau/basil/server/auth"
	"github.com/sambeau/basil/server/config"
)

// apiHandler handles API routes backed by Parsley modules.
type apiHandler struct {
	server     *Server
	route      config.Route
	scriptPath string
	cache      *scriptCache
}

func newAPIHandler(s *Server, route config.Route, cache *scriptCache) (*apiHandler, error) {
	scriptPath := route.Handler
	return &apiHandler{
		server:     s,
		route:      route,
		scriptPath: scriptPath,
		cache:      cache,
	}, nil
}

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	program, err := h.cache.getAST(h.scriptPath)
	if err != nil {
		h.server.logError("failed to load script: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	evaluator.ClearModuleCache()

	reqCtx := buildAPIRequestContext(r, h.route)

	env := evaluator.NewEnvironment()
	env.Filename = h.scriptPath

	// Set root path - distinguish between site mode and route mode
	var rootPath string
	scriptDir := filepath.Dir(h.scriptPath)
	absScriptDir, _ := filepath.Abs(scriptDir)

	if h.route.PublicDir != "" {
		absPublicDir, _ := filepath.Abs(h.route.PublicDir)
		// If handler is within or equal to PublicDir, use PublicDir as root (site mode)
		if strings.HasPrefix(absScriptDir+string(filepath.Separator), absPublicDir+string(filepath.Separator)) ||
			absScriptDir == absPublicDir {
			rootPath = absPublicDir
		} else {
			rootPath = absScriptDir
		}
	} else {
		rootPath = absScriptDir
	}
	env.RootPath = rootPath

	env.Security = &evaluator.SecurityPolicy{
		NoRead:        false,
		AllowWrite:    []string(h.server.config.Security.AllowWrite),
		AllowWriteAll: false,
		AllowExecute:  []string{rootPath},
		RestrictRead:  []string{"/etc", "/var", "/root"},
	}

	basilObj := buildBasilContext(r, h.route, reqCtx, h.server.db, h.server.dbDriver, h.route.PublicDir, h.server.fragmentCache, h.route.Path, "", nil)
	env.BasilCtx = basilObj

	// Set server-level database (available to modules at load time)
	if h.server.db != nil {
		env.ServerDB = evaluator.NewManagedDBConnection(h.server.db, h.server.dbDriver)
	}

	// Set fragment cache, asset registry, and handler path
	env.FragmentCache = h.server.fragmentCache
	env.AssetRegistry = h.server.assetRegistry
	env.AssetBundle = h.server.assetBundle
	env.BasilJSURL = JSAssetURL()
	env.HandlerPath = h.route.Path
	env.DevMode = h.server.config.Server.Dev

	// Inject publicUrl() function for asset registration
	env.SetProtected("publicUrl", evaluator.NewPublicURLBuiltin())

	// Inject @params - merged query+form params (POST wins)
	env.Set("@params", buildParams(r, env))

	if h.server.devLog != nil {
		env.DevLog = h.server.devLog
	}

	// Capture log() output for parity with page handlers
	scriptLogger := &scriptLogCapture{output: make([]string, 0)}
	env.Logger = scriptLogger

	result := evaluator.Eval(program, env)
	if result != nil && result.Type() == evaluator.ERROR_OBJ {
		errObj := result.(*evaluator.Error)
		h.server.logError("script error in %s: %s", h.scriptPath, errObj.Inspect())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	moduleDict := evaluator.ExportsToDict(env)
	subPath := strings.TrimPrefix(r.URL.Path, strings.TrimSuffix(h.route.Path, "/"))
	if subPath == "" {
		subPath = "/"
	}

	h.dispatchModule(w, r, moduleDict, subPath)
}

func (h *apiHandler) dispatchModule(w http.ResponseWriter, r *http.Request, module *evaluator.Dictionary, subPath string) {
	// Nested routing via `routes` export
	if routesObj := getModuleExport(module, "routes"); routesObj != nil {
		if routesDict, ok := routesObj.(*evaluator.Dictionary); ok {
			if nextModule, nextPath := matchRoute(routesDict, subPath); nextModule != nil {
				if dict, ok := nextModule.(*evaluator.Dictionary); ok {
					h.dispatchModule(w, r, dict, nextPath)
					return
				}
			}
		}
	}

	// Choose handler export
	hasID, idVal := extractID(subPath)
	exportName := mapMethodToExport(r.Method, hasID)

	handler := getModuleExport(module, exportName)
	if handler == nil {
		writeMethodNotAllowed(w, module)
		return
	}

	user, ok := h.enforceAuth(w, r, handler)
	if !ok {
		return // Response already written
	}

	if !h.enforceRateLimit(w, r, module, user) {
		return
	}

	reqObj := h.buildRequestObject(module.Env, r, idVal, user)
	result := evaluator.CallWithEnv(handler, []evaluator.Object{reqObj}, module.Env)

	if errObj, ok := result.(*evaluator.Error); ok {
		if errObj.UserDict != nil {
			h.writeUnifiedError(w, errObj)
			return
		}
		h.server.logError("runtime error in %s: %s", h.scriptPath, errObj.Inspect())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.writeAPIResponse(w, result)
}

// buildAPIRequestContext mirrors buildRequestContext but adds params when present.
func buildAPIRequestContext(r *http.Request, route config.Route) map[string]any {
	ctx := buildRequestContext(r, route)
	// params will be populated later when building the request object
	ctx["params"] = map[string]any{}
	return ctx
}

func (h *apiHandler) buildRequestObject(env *evaluator.Environment, r *http.Request, id string, user *auth.User) evaluator.Object {
	ctx := buildRequestContext(r, h.route)
	params := map[string]any{}
	if id != "" {
		params["id"] = id
	}
	ctx["params"] = params

	if user != nil {
		userMap := map[string]any{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		}
		ctx["user"] = userMap
	}

	obj, err := parsley.ToParsley(ctx)
	if err != nil {
		return evaluator.NULL
	}

	// Attach environment so nested evals (req.params) work as expected
	if dict, ok := obj.(*evaluator.Dictionary); ok {
		dict.Env = env
	}

	return obj
}

func (h *apiHandler) enforceAuth(w http.ResponseWriter, r *http.Request, handler evaluator.Object) (*auth.User, bool) {
	meta := readAuthMetadata(handler)

	user := auth.GetUser(r)

	if meta.AuthType == "public" {
		return user, true
	}

	if user == nil {
		h.writeUnifiedError(w, evaluator.ApiFailError("HTTP-401", "Unauthorized", http.StatusUnauthorized))
		return nil, false
	}

	// Role enforcement: check user's role against required roles
	if meta.AuthType == "admin" {
		if user.Role != auth.RoleAdmin {
			h.writeUnifiedError(w, evaluator.ApiFailError("HTTP-403", "Forbidden: admin role required", http.StatusForbidden))
			return nil, false
		}
	}
	if meta.AuthType == "roles" && len(meta.Roles) > 0 {
		if !sliceContains(meta.Roles, user.Role) {
			h.writeUnifiedError(w, evaluator.ApiFailError("HTTP-403", "Forbidden: insufficient role", http.StatusForbidden))
			return nil, false
		}
	}

	return user, true
}

func (h *apiHandler) enforceRateLimit(w http.ResponseWriter, r *http.Request, module *evaluator.Dictionary, user *auth.User) bool {
	limit, window := h.getRateLimit(module)
	key := rateLimitKey(r, user)
	if !h.server.rateLimiter.Allow(key, limit, window) {
		h.writeUnifiedError(w, evaluator.ApiFailError("HTTP-429", "Too Many Requests", http.StatusTooManyRequests))
		return false
	}
	return true
}

// sliceContains checks if a slice contains a string
func sliceContains(slice []string, s string) bool {
	return slices.Contains(slice, s)
}

func rateLimitKey(r *http.Request, user *auth.User) string {
	if user != nil && user.ID != "" {
		return "user:" + user.ID
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "ip:" + r.RemoteAddr
	}
	return "ip:" + host
}

func (h *apiHandler) getRateLimit(module *evaluator.Dictionary) (int, time.Duration) {
	limit := 60
	window := time.Minute

	rlObj := getModuleExport(module, "rateLimit")
	if rlObj == nil {
		return limit, window
	}

	rlDict, ok := rlObj.(*evaluator.Dictionary)
	if !ok {
		return limit, window
	}

	if reqExpr, ok := rlDict.Pairs["requests"]; ok {
		if iv, ok := evaluator.Eval(reqExpr, rlDict.Env).(*evaluator.Integer); ok {
			if iv.Value > 0 {
				limit = int(iv.Value)
			}
		}
	}

	if winExpr, ok := rlDict.Pairs["window"]; ok {
		val := evaluator.Eval(winExpr, rlDict.Env)
		switch w := val.(type) {
		case *evaluator.String:
			if d, err := time.ParseDuration(w.Value); err == nil && d > 0 {
				window = d
			}
		case *evaluator.Integer:
			if w.Value > 0 {
				window = time.Duration(w.Value) * time.Second
			}
		}
	}

	return limit, window
}

func readAuthMetadata(handler evaluator.Object) authMetadata {
	if wrapped, ok := handler.(*evaluator.AuthWrappedFunction); ok {
		return authMetadata{AuthType: wrapped.AuthType, Roles: wrapped.Roles}
	}
	return authMetadata{AuthType: "auth"}
}

type authMetadata struct {
	AuthType string
	Roles    []string
}

func getModuleExport(module *evaluator.Dictionary, name string) evaluator.Object {
	expr, ok := module.Pairs[name]
	if !ok {
		return nil
	}
	return evaluator.Eval(expr, module.Env)
}

func mapMethodToExport(method string, hasID bool) string {
	switch method {
	case http.MethodGet:
		if hasID {
			return "getById"
		}
		return "get"
	case http.MethodPost:
		return "post"
	case http.MethodPut:
		return "put"
	case http.MethodPatch:
		return "patch"
	case http.MethodDelete:
		return "delete"
	default:
		return ""
	}
}

func extractID(subPath string) (bool, string) {
	trimmed := strings.Trim(subPath, "/")
	if trimmed == "" {
		return false, ""
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 1 {
		return true, parts[0]
	}
	return false, ""
}

func matchRoute(routes *evaluator.Dictionary, subPath string) (evaluator.Object, string) {
	trimmed := "/" + strings.Trim(strings.TrimPrefix(subPath, "/"), "/")
	bestLen := -1
	var bestVal evaluator.Object
	var bestRest string

	for key, expr := range routes.Pairs {
		// Keys are stored as expressions; evaluate the key literal name
		routePath := key
		if after, ok := strings.CutPrefix(trimmed, routePath); ok {
			if len(routePath) > bestLen {
				bestLen = len(routePath)
				bestVal = evaluator.Eval(expr, routes.Env)
				bestRest = after
				if bestRest == "" {
					bestRest = "/"
				}
			}
		}
	}

	return bestVal, bestRest
}

func writeMethodNotAllowed(w http.ResponseWriter, module *evaluator.Dictionary) {
	allow := collectAllowedMethods(module)
	if len(allow) > 0 {
		sort.Strings(allow)
		w.Header().Set("Allow", strings.Join(allow, ", "))
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func collectAllowedMethods(module *evaluator.Dictionary) []string {
	methods := make(map[string]bool)
	for key := range module.Pairs {
		switch key {
		case "get":
			methods[http.MethodGet] = true
		case "getById":
			methods[http.MethodGet] = true
		case "post":
			methods[http.MethodPost] = true
		case "put":
			methods[http.MethodPut] = true
		case "patch":
			methods[http.MethodPatch] = true
		case "delete":
			methods[http.MethodDelete] = true
		}
	}

	allow := make([]string, 0, len(methods))
	for m := range methods {
		allow = append(allow, m)
	}
	return allow
}

func (h *apiHandler) writeAPIResponse(w http.ResponseWriter, obj evaluator.Object) {
	if obj == nil || obj == evaluator.NULL {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch v := obj.(type) {
	case *evaluator.Dictionary:
		status := http.StatusOK
		body := evaluator.Object(v)

		if statusExpr, ok := v.Pairs["status"]; ok {
			if iv, ok := evaluator.Eval(statusExpr, v.Env).(*evaluator.Integer); ok {
				status = int(iv.Value)
			}
		}

		if headersExpr, ok := v.Pairs["headers"]; ok {
			if headersDict, ok := evaluator.Eval(headersExpr, v.Env).(*evaluator.Dictionary); ok {
				for hk, hv := range headersDict.Pairs {
					if hvObj, ok := evaluator.Eval(hv, headersDict.Env).(*evaluator.String); ok {
						w.Header().Set(hk, hvObj.Value)
					}
				}
			}
		}

		if bodyExpr, ok := v.Pairs["body"]; ok {
			body = evaluator.Eval(bodyExpr, v.Env)
		}

		h.writeAsJSONOrText(w, status, body)
		return
	case *evaluator.Array:
		h.writeAsJSONOrText(w, http.StatusOK, v)
		return
	case *evaluator.String:
		contentType := "text/plain; charset=utf-8"
		if looksLikeHTML(v.Value) {
			contentType = "text/html; charset=utf-8"
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(v.Value))
		return
	default:
		h.writeAsJSONOrText(w, http.StatusOK, obj)
	}
}

func (h *apiHandler) writeUnifiedError(w http.ResponseWriter, err *evaluator.Error) {
	status := 500
	if err.UserDict != nil {
		if statusExpr, ok := err.UserDict.Pairs["status"]; ok {
			if iv, ok := evaluator.Eval(statusExpr, err.UserDict.Env).(*evaluator.Integer); ok {
				status = int(iv.Value)
			}
		}
	}

	// Wrap in {error: <dict>} envelope
	errorDict := err.UserDict
	if errorDict == nil {
		errorDict = evaluator.NewDictionaryFromObjectsWithOrder(
			map[string]evaluator.Object{
				"code":    &evaluator.String{Value: err.Code},
				"message": &evaluator.String{Value: err.Message},
			},
			[]string{"code", "message"},
		)
	}
	envelope := evaluator.NewDictionaryFromObjectsWithOrder(
		map[string]evaluator.Object{"error": errorDict},
		[]string{"error"},
	)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	h.writeJSONDict(w, envelope)
}

func (h *apiHandler) writeAsJSONOrText(w http.ResponseWriter, status int, obj evaluator.Object) {
	// Strings get special handling to allow plain text responses
	if s, ok := obj.(*evaluator.String); ok {
		contentType := "text/plain; charset=utf-8"
		if looksLikeHTML(s.Value) {
			contentType = "text/html; charset=utf-8"
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		w.Write([]byte(s.Value))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	h.writeJSON(w, obj)
}

func (h *apiHandler) writeJSON(w http.ResponseWriter, obj evaluator.Object) {
	goVal := parsley.FromParsley(obj)
	data, err := json.Marshal(goVal)
	if err != nil {
		h.server.logError("failed to marshal JSON: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func (h *apiHandler) writeJSONDict(w http.ResponseWriter, dict *evaluator.Dictionary) {
	goVal := parsley.FromParsley(dict)
	data, err := json.Marshal(goVal)
	if err != nil {
		h.server.logError("failed to marshal JSON: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func looksLikeHTML(s string) bool {
	trimmed := strings.TrimSpace(s)
	return strings.HasPrefix(trimmed, "<")
}
