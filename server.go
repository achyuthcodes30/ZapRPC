package zaprpc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"reflect"

	"github.com/quic-go/quic-go"
)

type ZapServer struct {
	services map[string]interface{}
	listener *quic.Listener
}

func NewZapServer() *ZapServer {
	log.Printf("Made new ZapServer!")
	return &ZapServer{
		services: make(map[string]interface{}),
	}
}

func (s *ZapServer) RegisterService(name string, service interface{}) {
	s.services[name] = service
	log.Printf("Registered service: %v", name)
}

func (s *ZapServer) Serve(port int, config *ZapConfig) error {
	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: port})
	if err != nil {
		return err
	}
	var tr *quic.Transport
	if config != nil {
		config.transportConfig.Conn = udpConn
		tr = config.transportConfig

	} else {
		tr = &quic.Transport{
			Conn: udpConn,
		}
	}
	var ln *quic.Listener

	if config != nil {
		if config.tlsConfig != nil {
			ln, err = tr.Listen(config.tlsConfig, config.quicConfig)
			if err != nil {
				return err
			}
		} else {
			ln, err = tr.Listen(generateTLSConfig(), nil)
			if err != nil {
				return err
			}

		}
	} else {
		ln, err = tr.Listen(generateTLSConfig(), nil)
		if err != nil {
			return err
		}

	}
	s.listener = ln
	log.Printf("listening on %v", port)
	for {
		var conn quic.Connection
		conn, err = ln.Accept(context.Background())
		if err != nil {
			return err
		}
		go s.handleSession(conn)

	}

}

func (s *ZapServer) handleSession(conn quic.Connection) {
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			log.Printf("Error accepting stream: %v", err)
			return
		}
		go s.handleStream(stream)
	}
}

func (s *ZapServer) handleStream(stream quic.Stream) {
	defer stream.Close()
	decoder := gob.NewDecoder(stream)
	encoder := gob.NewEncoder(stream)
	for {
		var request struct {
			ServiceMethod string
			Args          []interface{}
		}
		err := decoder.Decode(&request)
		if err != nil {
			log.Printf("Error decoding request: %v", err)
			return
		}

		response, err := s.callMethod(request.ServiceMethod, request.Args)
		if err != nil {
			log.Printf("Error calling method: %v", err)
			encoder.Encode(ZapValue{Value: struct{ Error string }{err.Error()}})
			continue
		}
		err = encoder.Encode(ZapValue{Value: response})
		if err != nil {
			log.Printf("Error encoding response: %v", err)
			return
		}
	}
}

func (s *ZapServer) callMethod(serviceMethod string, args []interface{}) (interface{}, error) {
	serviceName, methodName, found := parseServiceMethod(serviceMethod)
	if !found {
		return nil, fmt.Errorf("invalid service method: %s", serviceMethod)
	}

	service, ok := s.services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	method := reflect.ValueOf(service).MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method not found: %s", methodName)
	}

	reflectArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		reflectArgs[i] = reflect.ValueOf(arg)
	}

	results := method.Call(reflectArgs)

	// If the method returns an error, it should be the last return value
	if len(results) > 0 {
		lastResult := results[len(results)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !lastResult.IsNil() {
				return nil, lastResult.Interface().(error)
			}
			results = results[:len(results)-1] // Remove the error from results
		}
	}

	// If there's only one result (excluding a potential error), return it directly
	if len(results) == 1 {
		return results[0].Interface(), nil
	}

	// If there are multiple results, return them as a slice
	response := make([]interface{}, len(results))
	for i, result := range results {
		response[i] = result.Interface()
	}

	return response, nil
}

func parseServiceMethod(serviceMethod string) (string, string, bool) {
	for i := 0; i < len(serviceMethod); i++ {
		if serviceMethod[i] == '.' {
			return serviceMethod[:i], serviceMethod[i+1:], true
		}
	}
	return "", "", false
}

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"new-zap"},
	}
}
