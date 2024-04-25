// Implements the helloworld greeter GRPC server, including a multiplexed GRPC
// gateway so that serving on a single port can handle both JSON/HTTP requests
// via HTTP1 and RPC calls via HTTP2.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"boiler.plate/services/greeter"

	"github.com/NYTimes/gziphandler"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	// Binding to a specific hostname that points to 127.0.0.1 can be helpful in
	// preventing mac security warnings during development.
	host = "localhost"
	port = 5050

	// Demonstrating using self-signed certs for development.
	certFile = "gen/certs/server.crt"
	keyFile  = "gen/certs/server.key"

	// Serves an index.html which can be used for testing locally.
	staticFiles = "gen/www/"
)

func main() {

	endpoint := fmt.Sprintf("%s:%d", host, port)

	// Configure GRPC internal logging. This is noisy and you will likely want to
	// implement your own log adaptor that uses structured outputs.
	// ===========================================================================
	log := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)
	grpclog.SetLoggerV2(log)

	// Set up the gRPC server.
	// ===========================================================================
	log.Info("⚙️  Setting up GRPC server")
	grpcServer := grpc.NewServer(
		// You'll probably want a different way of getting/loading credentials.
		grpc.Creds(serverTLSFromFile(certFile, keyFile)),

		// Sample middleware. Write your own interceptors or find useful pre-built
		// ones here: https://github.com/grpc-ecosystem/go-grpc-middleware
		grpc.ChainUnaryInterceptor(
			grpcLogger(log),
			recovery.UnaryServerInterceptor(),
		),
	)

	// Add the additional services you create here.
	greeter.RegisterGreeterServer(grpcServer, greeter.New(log))

	// Create the GRPC Gateway and register the handlers.
	// ===========================================================================
	log.Info("⚙️  Configuring GRPC Gateway")

	gateway := runtime.NewServeMux(
		// Override default JSON marshaler so that 0, false, and "" are emitted as
		// actual values rather than undefined. This allows for better handling of
		// PB wrapper types that allow for true, false, null.
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				Multiline:       true,
				Indent:          "  ",
				EmitUnpopulated: true,
			},
		}),

		// Set up error handling for logging, analytics, or custom error responses.
		runtime.WithErrorHandler(errorHandler(log)),

		// Set up the GRPC Gateway to handle form encoded data.
		runtime.WithMarshalerOption("application/x-www-form-urlencoded", &formDecoder{}),

		// You'll likely find you want to do some custom mapping of HTTP to GRPC
		// headers, if so override these two methods.
		runtime.WithIncomingHeaderMatcher(runtime.DefaultHeaderMatcher),
		runtime.WithOutgoingHeaderMatcher(runtime.DefaultHeaderMatcher),
	)

	opts := []grpc.DialOption{grpc.WithTransportCredentials(clientTLSFromFile(certFile))}

	// Add the additional services you create here.
	if err := greeter.RegisterGreeterHandlerFromEndpoint(context.Background(), gateway, endpoint, opts); err != nil {
		panic(err)
	}

	// Set up a server to receive HTTP traffic.
	// ===========================================================================
	log.Info("⚙️  Setting up HTTP server")
	httpMux := http.NewServeMux()

	// Arbitrary handlers can be added here. They should come before the GRPC Gateway.
	httpMux.HandleFunc("/robots.txt", robots)

	// The GRPC Gateway, if you change the API prefix you'll need to update this.
	httpMux.Handle("/v1/", gateway)

	// This static file server should be removed or hardened from a prod app.
	httpMux.Handle("/", http.FileServer(http.Dir(staticFiles)))

	// Chain handlers for logging and gzip.
	httpHandler := httpLogger(log, gziphandler.GzipHandler(httpMux))

	// Create a server that multiplexes HTTP and GRPC traffic.
	// ===========================================================================
	log.Info("⚙️  Setting up multiplexer")
	multiplexer := &http.Server{
		Addr:      endpoint,
		Handler:   multiplex(grpcServer, httpHandler),
		TLSConfig: safeTLSConfig(),
	}

	log.Infof("🚀 Listening for traffic on %s", endpoint)
	multiplexer.ListenAndServeTLS(certFile, keyFile)
}

// Sample interceptor that logs the GRPC method being called.
func grpcLogger(log grpclog.LoggerV2) func(context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log.Infof("✨ GRPC Request : %s", info.FullMethod)
		return handler(ctx, req)
	}
}

// Simple HTTP logging handler.
func httpLogger(log grpclog.LoggerV2, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Infof("⚡️ HTTP/%d Request : %s", req.ProtoMajor, req.URL)
		handler.ServeHTTP(w, req)
	})
}

func errorHandler(log grpclog.LoggerV2) runtime.ErrorHandlerFunc {
	return func(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, req *http.Request, err error) {
		log.Errorf("😬 Gateway error (%s) : %v", req.URL, err)
		runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, req, err)
	}
}

// HTTP handler function that writes a minimal robots.txt file that prevents indexing.
func robots(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("User-agent: *\nDisallow: /\n"))
}

// Creates credentials from a cert and key file, while using restricted ciphers.
// See https://github.com/grpc/grpc-go/blob/7aea499f9110a479b3777df064372f44188146aa/credentials/credentials.go#L212
func serverTLSFromFile(cert, key string) credentials.TransportCredentials {
	c, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		panic(err)
	}
	tlsConfig := safeTLSConfig()
	tlsConfig.Certificates = []tls.Certificate{c}
	return credentials.NewTLS(tlsConfig)
}

// See https://github.com/grpc/grpc-go/blob/7aea499f9110a479b3777df064372f44188146aa/credentials/credentials.go#L195
func clientTLSFromFile(cert string) credentials.TransportCredentials {
	b, err := os.ReadFile(cert)
	if err != nil {
		panic(err)
	}
	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(b) {
		panic("Failed to append credentials")
	}
	tlsConfig := safeTLSConfig()
	tlsConfig.RootCAs = cp
	return credentials.NewTLS(tlsConfig)
}

// safeTLSConfig returns a restrictive TLS config intended to be safe for HIPAA
// and other sensitive data. You may need to adjust this for your own needs.
func safeTLSConfig() *tls.Config {
	return &tls.Config{
		NextProtos: []string{"h2"},
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		SessionTicketsDisabled: true,
	}
}

func multiplex(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}

// Forked from pending issue here:
// https://github.com/grpc-ecosystem/grpc-gateway/issues/7
type formDecoder struct {
	runtime.Marshaler
}

// ContentType means the content type of the response
func (u formDecoder) ContentType(_ interface{}) string {
	return "application/json"
}

func (u formDecoder) Marshal(v interface{}) ([]byte, error) {
	// can marshal the response in proto message format
	j := runtime.JSONPb{}
	return j.Marshal(v)
}

// NewDecoder indicates how to decode the request
func (u formDecoder) NewDecoder(r io.Reader) runtime.Decoder {
	return runtime.DecoderFunc(func(p interface{}) error {
		msg, ok := p.(proto.Message)
		if !ok {
			return fmt.Errorf("not proto message")
		}

		formData, err := io.ReadAll(r)
		if err != nil {
			return err
		}

		values, err := url.ParseQuery(string(formData))
		if err != nil {
			return err
		}

		filter := &utilities.DoubleArray{}
		err = runtime.PopulateQueryParameters(msg, values, filter)
		if err != nil {
			return err
		}
		return nil
	})
}
