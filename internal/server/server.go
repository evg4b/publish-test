package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/pterm/pterm"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	http     http.Server
	httpPort int

	https     http.Server
	httpsPort int
	cert      string
	key       string

	handler func(http.ResponseWriter, *http.Request)
}

const addr = "0.0.0.0"

const (
	defaultHTTPPort  = 80
	defaultHTTPSPort = 443
)

func NewServer(options ...serverOption) *Server {
	appServer := &Server{
		httpPort:  defaultHTTPPort,
		httpsPort: defaultHTTPSPort,
	}

	for _, option := range options {
		option(appServer)
	}

	address := net.JoinHostPort(addr, strconv.Itoa(appServer.httpPort))
	appServer.http = http.Server{
		Addr: address,
		Handler: http.HandlerFunc(
			infrastructure.NormalizeHTTPReqDecorator(appServer.handler),
		),
	}

	if appServer.isHTTPSAvialable() {
		address = net.JoinHostPort(addr, strconv.Itoa(appServer.httpPort))
		appServer.https = http.Server{
			Addr: address,
			Handler: http.HandlerFunc(
				infrastructure.NormalizeHTTPSReqDecorator(appServer.handler),
			),
		}
	}

	return appServer
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	rungroup, ctx := errgroup.WithContext(ctx)

	rungroup.Go(func() error {
		if err := s.http.ListenAndServe(); err != nil {
			pterm.Fatal.Println(err)
		}

		return nil
	})

	if s.isHTTPSAvialable() {
		rungroup.Go(func() error {
			if err := s.https.ListenAndServeTLS(s.cert, s.key); err != nil {
				pterm.Fatal.Println(err)
			}

			return nil
		})
	}

	rungroup.Go(func() error {
		<-ctx.Done()
		if err := s.http.Shutdown(context.Background()); isSucessClosed(err) {
			return fmt.Errorf("shutdown http server %w", err)
		}

		return nil
	})

	if s.isHTTPSAvialable() {
		rungroup.Go(func() error {
			<-ctx.Done()

			if err := s.http.Shutdown(context.Background()); isSucessClosed(err) {
				return fmt.Errorf("shutdown http server %w", err)
			}

			return nil
		})
	}

	return rungroup.Wait()
}

func (s *Server) isHTTPSAvialable() bool {
	return len(s.cert) > 0 && len(s.key) > 0
}

func isSucessClosed(err error) bool {
	return err != nil && !errors.Is(err, http.ErrServerClosed)
}
