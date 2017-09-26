package config

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// MakeHandler returns an http.Handler for the config service.
func MakeHandler(logger log.Logger, svc ServiceUser) http.Handler {
	r := mux.NewRouter()
	r.StrictSlash(true)

	r.Methods("PUT").Path(`/{baseConfig:[a-z0-9]+}`).Name("configUser").Handler(
		kithttp.NewServer(
			userEndpoint(svc),
			decodeUserRequest,
			kithttp.EncodeJSONResponse,
		),
	)

	return r
}

func decodeUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	c, ok := mux.Vars(r)["baseConfig"]
	if !ok {
		return nil, fmt.Errorf("Baseconfig missing")
	}
	return userRequest{baseConfig: c}, nil
}
