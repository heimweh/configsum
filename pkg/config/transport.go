package config

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/xeipuuv/gojsonschema"

	"github.com/lifesum/configsum/pkg/errors"
)

// Headers.
const (
	headerContentType = "Content-Type"
	headerBaseID      = "X-Configsum-Base-Id"
	headerBaseName    = "X-Configsum-Base-Name"
	headerClientID    = "X-Configsum-Client-Id"
	headerID          = "X-Configsum-Id"
	headerCreatedAt   = "X-Configsum-Created"
)

// URL fragments.
const (
	varBaseConfig muxVar = "baseConfig"
)

type muxVar string

// MakeHandler returns an http.Handler for the config service.
func MakeHandler(
	svc ServiceUser,
	auth endpoint.Middleware,
	opts ...kithttp.ServerOption,
) http.Handler {
	r := mux.NewRouter()
	r.StrictSlash(true)

	r.Methods("PUT").Path(`/{baseConfig:[a-z0-9\-]+}`).Name("configUser").Handler(
		kithttp.NewServer(
			auth(userEndpoint(svc)),
			decodeJSONSchema(decodeUserRequest, decodeClientPayloadSchema),
			encodeUserResponse,
			append(
				opts,
				kithttp.ServerBefore(extractMuxVars(varBaseConfig)),
			)...,
		),
	)

	return r
}

func decodeJSONSchema(
	next kithttp.DecodeRequestFunc,
	schema *gojsonschema.Schema,
) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		raw, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}

		if len(raw) == 0 {
			return nil, errors.Wrap(errors.ErrInvalidPayload, "empty body")
		}
		res, err := schema.Validate(gojsonschema.NewBytesLoader(raw))
		if nil != err {
			return nil, errors.Wrap(errors.ErrInvalidPayload, err.Error())
		}

		if !res.Valid() {
			err := errors.ErrInvalidPayload

			for _, e := range res.Errors() {
				err = errors.Wrap(err, e.String())
			}

			return nil, err
		}

		r.Body = ioutil.NopCloser(bytes.NewBuffer(raw))

		return next(ctx, r)
	}
}

func extractMuxVars(keys ...muxVar) kithttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		for _, k := range keys {
			if v, ok := mux.Vars(r)[string(k)]; ok {
				ctx = context.WithValue(ctx, k, v)
			}
		}

		return ctx
	}
}

func decodeUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	baseConfig, ok := ctx.Value(varBaseConfig).(string)
	if !ok {
		return nil, errors.Wrap(errors.ErrVarMissing, "baseConfig missing")
	}

	c := userContext{}

	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		return nil, err
	}

	return userRequest{
		baseConfig: baseConfig,
		context:    c,
	}, nil
}

func encodeUserResponse(
	_ context.Context,
	w http.ResponseWriter,
	response interface{},
) error {
	r := response.(userResponse)

	w.Header().Set(headerContentType, "application/json; charset=utf-8")
	w.Header().Set(headerBaseID, r.baseID)
	w.Header().Set(headerBaseName, r.baseName)
	w.Header().Set(headerClientID, r.clientID)
	w.Header().Set(headerID, r.id)
	w.Header().Set(headerCreatedAt, r.createdAt.Format(time.RFC3339Nano))

	return json.NewEncoder(w).Encode(r.rendered)
}
