package middleware

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/achilleasa/usrv"
)

// Given a handler method that returns `error` and accepts two pointer arguments,
// each of the arguments pointing to a user-defined structure that defines the
// format for the request and response messages, this function will generate a
// usrv.Handler that automatically unmarshals incoming message payloads to
// the expected request structure, invokes the handler method and then marshals
// back the response or the error (if a non-nil error value is returned) to the
// usrv response message.
//
// The generated handler will also catch any panic() invocations from within
// the user handler and return them as errors if the recoverFromPanic argument
// is set to true.
func JsonHandler(handler interface{}, recoverFromPanic bool) usrv.Handler {
	// Analyze the handler args  using reflection
	typeData := reflect.TypeOf(handler)

	if typeData.Kind() != reflect.Func ||
		typeData.NumIn() != 2 ||
		typeData.In(0).Kind() != reflect.Ptr ||
		typeData.In(1).Kind() != reflect.Ptr ||
		typeData.NumOut() != 1 ||
		typeData.Out(0).Name() != "error" {
		panic("Argument signature must be a function receiving two pointer arguments to the request and response structs and return error")
	}

	handlerFn := reflect.ValueOf(handler)

	// Fetch real type data for the args
	reqType := typeData.In(0).Elem()
	resType := typeData.In(1).Elem()

	return func(req usrv.Message, res usrv.Message) {
		if recoverFromPanic {
			defer func() {
				if err := recover(); err != nil {
					if e, ok := err.(error); ok {
						res.SetContent(nil, e)
					} else {
						res.SetContent(nil, errors.New(err.(string)))
					}
				}
			}()
		}

		// Unserialize request
		reqContent, _ := req.Content()
		reqObj := reflect.New(reqType)
		err := json.Unmarshal(reqContent, reqObj.Interface())
		if err != nil {
			res.SetContent(nil, err)
			return
		}
		resObj := reflect.New(resType)

		// Invoke handler
		retVals := handlerFn.Call([]reflect.Value{reqObj, resObj})
		ret := retVals[0].Interface()
		if ret != nil {
			res.SetContent(nil, ret.(error))
			return
		}

		// Serialize back to response
		resBytes, err := json.Marshal(resObj.Interface())
		if err != nil {
			res.SetContent(nil, err)
			return
		}
		res.SetContent(resBytes, nil)
	}
}
