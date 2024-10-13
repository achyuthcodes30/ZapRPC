package zaprpc

import (
	"context"
	"crypto/tls"
	"encoding/gob"
	"fmt"

	"github.com/quic-go/quic-go"
)

func NewConn(ctx context.Context, target string, config *ZapConfig) (quic.Connection, error) {
	var conn quic.Connection
	var err error
	if config != nil {
		if config.tlsConfig != nil {
			conn, err = quic.DialAddr(ctx, target, config.tlsConfig, config.quicConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to dial: %w", err)
			}
		} else {
			var tlsConf = &tls.Config{
				InsecureSkipVerify: true,
				NextProtos:         []string{"new-zap"},
			}
			conn, err = quic.DialAddr(ctx, target, tlsConf, config.quicConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to dial: %w", err)
			}

		}

	} else {
		var tlsConf = &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"new-zap"},
		}
		conn, err = quic.DialAddr(ctx, target, tlsConf, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to dial without config: %w", err)
		}

	}
	return conn, nil
}

func Zap(ctx context.Context, conn quic.Connection, serviceMethod string, args ...interface{}) (interface{}, error) {
	stream, err := conn.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()
	decoder := gob.NewDecoder(stream)
	encoder := gob.NewEncoder(stream)
	request := struct {
		ServiceMethod string
		Args          []interface{}
	}{
		ServiceMethod: serviceMethod,
		Args:          args,
	}

	err = encoder.Encode(request)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}
	var response ZapValue
	err = decoder.Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if err, ok := response.Value.(struct{ Error string }); ok && err.Error != "" {
		return nil, fmt.Errorf(err.Error)
	}

	return response.Value, nil
}

/* func NewZapServiceClient[T any](ctx context.Context, conn quic.Connection, service string) (T, error) {
	proxy := reflect.ValueOf(new(T))
	for i := 0; i < proxy.NumMethod(); i++ {
		method := proxy.Type().Method(i)
		proxy.Method(i).Set(reflect.MakeFunc(method.Type, func(args []reflect.Value) []reflect.Value {
			callArgs := make([]interface{}, len(args)-1)
			for i, arg := range args[1:] {
				callArgs[i] = arg.Interface()
			}

			result, err := Zap(ctx, conn, service+"."+method.Name, callArgs...)
			if err != nil {
				// Assuming the last return value is an error
				returnValues := make([]reflect.Value, method.Type.NumOut())
				for i := 0; i < method.Type.NumOut()-1; i++ {
					returnValues[i] = reflect.Zero(method.Type.Out(i))
				}
				returnValues[method.Type.NumOut()-1] = reflect.ValueOf(&err).Elem()
				return returnValues
			}

			returnValues := make([]reflect.Value, method.Type.NumOut())
			resultValue := reflect.ValueOf(result)

			if resultValue.Kind() == reflect.Slice {
				for i := 0; i < method.Type.NumOut()-1; i++ {
					returnValues[i] = resultValue.Index(i).Convert(method.Type.Out(i))
				}
			} else if method.Type.NumOut() > 1 {
				returnValues[0] = resultValue.Convert(method.Type.Out(0))
			}

			// Set the error return value to nil
			returnValues[method.Type.NumOut()-1] = reflect.Zero(method.Type.Out(method.Type.NumOut() - 1))

			return returnValues
		}))
	}

	return proxy.Interface().(T), nil
} */
