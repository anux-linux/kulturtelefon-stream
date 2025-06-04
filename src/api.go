// StreamAPI initializes and returns a new instance of ApiServer.
// It sets up the storage and Icecast configuration based on the provided configuration.
//
// Parameters:
//   - listenAddr: The address on which the API server will listen for incoming requests.
//   - config: The configuration object containing necessary settings for the server.
//
// Returns:
//   - *ApiServer: A pointer to the initialized ApiServer instance.
//   - error: An error if the initialization fails, otherwise nil.
//
// This function performs the following steps:
//  1. Initializes the security key.
//  2. Creates a new SQLite storage instance using the provided configuration.
//  3. Creates a new Icecast configuration store.
//  4. Returns an ApiServer instance with the initialized components.
//
// If any of the steps fail, the function logs the error and returns a descriptive error message.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type apiFunc func(http.ResponseWriter, *http.Request) error

type ApiError struct {
	Error string `json:"error"`
}

type ApiServer struct {
	listenAddr string
	storage    Store
	icecast    *IcecastConfigStore
}

// routeRightsMap is a map that associates HTTP routes with their corresponding rights.
var routeRightsMap = make(map[string]string)

type Middleware func(http.Handler) http.HandlerFunc

func StreamAPI(listenAddr string, config Config) (*ApiServer, error) {

	storage, err := NewSqliteStore(&config)
	if err != nil {
		logWithCaller("Failed to create storage", FatalLog)
		return nil, fmt.Errorf("failed to create storage")
	}

	icecast := NewIcecastConfig(config)
	if icecast == nil {
		logWithCaller("Failed to create icecast config", FatalLog)
		return nil, fmt.Errorf("failed to create icecast config")
	}

	logWithCaller(fmt.Sprintf("Created storage and icecast config for server listening on %s", listenAddr), DebugLog)

	return &ApiServer{
		listenAddr: listenAddr,
		storage:    storage,
		icecast:    icecast,
	}, nil
}

func MiddlewareChain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next.ServeHTTP
	}
}

func requestLoggerMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logWithCaller(fmt.Sprintf("Request: %s %s", r.Method, r.URL.Path), InfoLog)
		next.ServeHTTP(w, r)
	}
}

func requireAuthMiddlware(next http.Handler, api *ApiServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if !autherized(token, r, api) {
			WriteJson(w, http.StatusUnauthorized, ApiError{Error: "Unauthorized"})
			return
		}
		logWithCaller("Authorized", InfoLog)
		next.ServeHTTP(w, r)

	}
}

// autherized checks if the request is authorized based on the provided token and request.
func autherized(token string, r *http.Request, api *ApiServer) bool {

	logWithCaller("Autherizing", InfoLog)

	requestURL := r.URL.Path
	//we are getting all the parts of the request URL to check which right to use
	requestURLParts := strings.Split(requestURL, "/")

	logWithCaller(fmt.Sprintf("Autherizing for %s", requestURL), InfoLog)

	if len(requestURLParts) > 4 {

		//Our URLs c3 or for parts /api/streams/{streamName}
		logWithCaller(fmt.Sprintf("Request URL to long. Conatins %d parts", len(requestURLParts)), DebugLog)
		return false
	} else if len(requestURLParts) == 4 {
		//Case the URL is /api/streams/{streamName}
		rightsKey := r.Method + " " + strings.Join(requestURLParts[:3], "/") + "/{streamName}"
		logWithCaller(fmt.Sprintf("Getting rights for this call %s", rightsKey), InfoLog)

		right := routeRightsMap[rightsKey]
		logWithCaller(fmt.Sprintf("Checking this right %s", right), DebugLog)
		username, err := api.storage.GetUserByToken(getHash(token))
		if err != nil {
			return false
		}
		logWithCaller(fmt.Sprintf("Checking for user %s and token %s: %s", username, getHash(token), right), DebugLog)
		return checkTokeHasRight(token, right, username)
	} else if len(requestURLParts) == 3 {
		//Case the URL is /api/streams
		rightsKey := r.Method + " " + requestURL
		logWithCaller(fmt.Sprintf("Getting rights for this call %s", rightsKey), InfoLog)

		right := routeRightsMap[rightsKey]
		logWithCaller(fmt.Sprintf("Checking this right %s", right), DebugLog)
		username, err := api.storage.GetUserByToken(getHash(token))
		if err != nil {
			return false
		}
		logWithCaller(fmt.Sprintf("Checking for user %s and token %s: %s", username, getHash(token), right), DebugLog)
		return checkTokeHasRight(token, right, username)

	}

	logWithCaller("Nothing matches", DebugLog)
	return false

}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err != nil {
			WriteJson(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}

func WriteJson(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func (s *ApiServer) Run() error {
	router := http.NewServeMux()

	middlewareChain := MiddlewareChain(
		requestLoggerMiddleware,
	)

	server := http.Server{
		Addr:    s.listenAddr,
		Handler: middlewareChain(router),
	}
	logWithCaller(fmt.Sprintf("Server listening on %s", server.Addr), DebugLog)

	s.addPublicRoutes(router)
	s.addAuthorizedRoutes(router)
	s.addUserRoutes(router)

	logWithCaller(fmt.Sprintf("Starting server on %s", s.listenAddr), InfoLog)
	return server.ListenAndServe()

}

// ###########
// Public routes
// ###########
func (s *ApiServer) addPublicRoutes(router *http.ServeMux) {
	publicRouter := http.NewServeMux()

	public := "/public/"
	publicRouter.HandleFunc("GET "+public+"health", makeHTTPHandleFunc(s.handleHealthCheck))
	publicRouter.HandleFunc("GET "+public+"version", makeHTTPHandleFunc(s.handleVersion))

	router.Handle(public, publicRouter)
	logWithCaller("Added public routes", InfoLog)
}

func (s *ApiServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) error {
	return WriteJson(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *ApiServer) handleVersion(w http.ResponseWriter, r *http.Request) error {
	return WriteJson(w, http.StatusOK, map[string]string{"version": "0.0.5"})
}

// ###########
// User routes
// ###########

func (s *ApiServer) addUserRoutes(router *http.ServeMux) {
	userRouter := http.NewServeMux()

	user := "/user/"

	userRouter.HandleFunc("GET "+user+"token", makeHTTPHandleFunc(s.handleGetToken))

	router.Handle(user, userRouter)
	logWithCaller("Added user routes", InfoLog)
}

// handleGetToken handles the request to get a token for a user.
// User is authenticated with username and password.
func (s *ApiServer) handleGetToken(w http.ResponseWriter, r *http.Request) error {
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")
	logWithCaller(fmt.Sprintf("Getting token for user %s", username), InfoLog)
	if username == "" || password == "" {
		return fmt.Errorf("invalid credentials")
	}

	logWithCaller(fmt.Sprintf("Checking Password for user %s", username), InfoLog)
	passwordHash, err := s.storage.GetUser(username)
	if err != nil {
		return fmt.Errorf("invalid credentials")
	}

	if !validPassword(password, passwordHash) {
		return fmt.Errorf("invalid credentials")
	}

	logWithCaller(fmt.Sprintf("Creating token for user %s", username), InfoLog)
	token, err := createToken(username, rightsAdmin, time.Now().Add(24*365*time.Hour))

	s.storage.SaveToken(username, getHash(token))

	if err != nil {
		return fmt.Errorf("error getting token: %s", err)
	}

	return WriteJson(w, http.StatusOK, map[string]string{"token": token})
}

// ###########
// Api routes
// ###########
func (s *ApiServer) addAuthorizedRoutes(router *http.ServeMux) {

	autherizedRouter := http.NewServeMux()
	autherized := "/api/"

	autherizedRouter.HandleFunc("POST "+autherized+"streams", makeHTTPHandleFunc(s.handleCreateStream))
	addToRouteRightsMap("POST "+autherized+"streams", "post_stream")

	autherizedRouter.HandleFunc("GET "+autherized+"streams", makeHTTPHandleFunc(s.handleGetAllStreams))
	addToRouteRightsMap("GET "+autherized+"streams", "get_all_streams")

	autherizedRouter.HandleFunc("GET "+autherized+"streams/{streamName}", makeHTTPHandleFunc(s.handleGetSingleStream))
	addToRouteRightsMap("GET "+autherized+"streams/{streamName}", "get_stream")

	autherizedRouter.HandleFunc("POST "+autherized+"streams/{streamName}", makeHTTPHandleFunc(s.handleUpdateStream))
	addToRouteRightsMap("POST "+autherized+"streams/{streamName}", "post_stream")

	autherizedRouter.HandleFunc("DELETE "+autherized+"streams/{streamName}", makeHTTPHandleFunc(s.handleDeleteStream))
	addToRouteRightsMap("DELETE "+autherized+"streams/{streamName}", "delete_stream")

	middlewareChain := MiddlewareChain(
		func(next http.Handler) http.HandlerFunc {
			return requireAuthMiddlware(next, s)
		},
	)

	router.Handle(autherized, middlewareChain(autherizedRouter))
	logWithCaller("Added authorized routes", InfoLog)
}

func addToRouteRightsMap(route, right string) {
	routeRightsMap[route] = right
}

func (s *ApiServer) handleCreateStream(w http.ResponseWriter, r *http.Request) error {
	var mount IcecastMount
	err := json.NewDecoder(r.Body).Decode(&mount)
	if err != nil {
		logWithCaller(fmt.Sprintf("JSON error updating icecast mount: %v %s", mount, err), WarnLog)
		return fmt.Errorf("invalid JSON")
	}
	defer r.Body.Close()

	err = s.storage.CreateIcecastMount(mount)
	if err != nil {
		logWithCaller(fmt.Sprintf("Database error creating icecast mount: %v %s", mount, err), WarnLog)
		return fmt.Errorf("database error")
	}

	err = s.icecast.SaveMountConfig(mount)
	if err != nil {
		logWithCaller(fmt.Sprintf("Config error creating icecast mount: %v %s", mount, err), WarnLog)
		return fmt.Errorf("file error")
	}

	return WriteJson(w, http.StatusCreated, mount)
}

func (s *ApiServer) handleGetAllStreams(w http.ResponseWriter, r *http.Request) error {
	mounts, err := s.storage.GetIcecastMounts()
	if err != nil {
		logWithCaller(fmt.Sprintf("Database error fetching icecast mounts: %s", err), WarnLog)
		return fmt.Errorf("database error")
	}

	return WriteJson(w, http.StatusOK, mounts)
}

func (s *ApiServer) handleGetSingleStream(w http.ResponseWriter, r *http.Request) error {
	mountName := r.PathValue("streamName")
	logWithCaller(fmt.Sprintf("Getting mount for mountName: %s", mountName), InfoLog)
	if mountName == "" {
		return fmt.Errorf("missing stream name")
	}

	mount, err := s.storage.GetIcecastMount(mountName)
	if err != nil {
		logWithCaller(fmt.Sprintf("Database error fetching icecast mount: %s %s", mountName, err), WarnLog)
		return fmt.Errorf("database error")
	}

	return WriteJson(w, http.StatusOK, mount)
}
func (s *ApiServer) handleUpdateStream(w http.ResponseWriter, r *http.Request) error {
	mountName := r.PathValue("streamName")
	if mountName == "" {
		return fmt.Errorf("missing stream name")
	}
	logWithCaller(fmt.Sprintf("Updating stream %s", mountName), InfoLog)

	var mount IcecastMount
	err := json.NewDecoder(r.Body).Decode(&mount)
	if err != nil {
		logWithCaller(fmt.Sprintf("JSON error updating icecast mount: %s %s", mountName, err), WarnLog)
		return fmt.Errorf("invalid JSON")
	}
	defer r.Body.Close()

	mount.MountName = mountName

	err = s.storage.UpdateIcecastMount(mount)
	if err != nil {
		logWithCaller(fmt.Sprintf("Database error updating icecast mount: %s %s", mountName, err), WarnLog)
		return fmt.Errorf("database error")
	}

	err = s.icecast.SaveMountConfig(mount)
	if err != nil {
		logWithCaller(fmt.Sprintf("Config error creating icecast mount: %s %s", mountName, err), WarnLog)
		return fmt.Errorf("file error")
	}

	return WriteJson(w, http.StatusOK, mount)
}
func (s *ApiServer) handleDeleteStream(w http.ResponseWriter, r *http.Request) error {

	mountName := r.PathValue("streamName")
	if mountName == "" {
		return fmt.Errorf("missing stream name")
	}
	logWithCaller(fmt.Sprintf("Deleting stream %s", mountName), InfoLog)

	mount, err := s.storage.DeleteIcecastMount(mountName)
	if err != nil {
		logWithCaller(fmt.Sprintf("Database error deleting icecast mount: %s %s", mountName, err), WarnLog)
		return fmt.Errorf("database error")
	}

	err = s.icecast.DeleteMountConfig(mount)
	if err != nil {
		logWithCaller(fmt.Sprintf("Config error creating icecast mount: %s %s", mountName, err), WarnLog)
		return fmt.Errorf("file error")
	}

	return WriteJson(w, http.StatusOK, map[string]string{"status": "deleted"})
}
