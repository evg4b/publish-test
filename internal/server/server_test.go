package server_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNewUncorsServer(t *testing.T) {
	ctx := context.Background()
	expectedResponse := "UNCORS OK!"

	var handler http.HandlerFunc = func(w http.ResponseWriter, _r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, expectedResponse)
		if err != nil {
			panic(err)
		}
	}

	t.Run("handle request", func(t *testing.T) {
		t.Run("HTTP", func(t *testing.T) {
			uncorsServer := server.NewUncorsServer(ctx, handler)
			defer func() {
				err := uncorsServer.Close()
				testutils.CheckNoServerError(t, err)
			}()

			go func() {
				err := uncorsServer.ListenAndServe("127.0.0.1:0")
				testutils.CheckNoServerError(t, err)
			}()

			time.Sleep(300 * time.Millisecond)
			uri, err := url.Parse("http://" + uncorsServer.Addr)
			testutils.CheckNoError(t, err)

			res, err := http.DefaultClient.Do(&http.Request{URL: uri, Method: http.MethodGet})
			testutils.CheckNoError(t, err)
			defer func() {
				testutils.CheckNoError(t, res.Body.Close())
			}()

			data, err := io.ReadAll(res.Body)
			testutils.CheckNoError(t, err)

			assert.Equal(t, expectedResponse, string(data))
		})

		t.Run("HTTPS", testutils.WithTmpCerts(func(t *testing.T, certs *testutils.Certs) {
			uncorsServer := server.NewUncorsServer(ctx, handler)
			defer func() {
				testutils.CheckNoServerError(t, uncorsServer.Close())
			}()

			go func() {
				err := uncorsServer.ListenAndServeTLS("127.0.0.1:0", certs.CertPath, certs.KeyPath)
				testutils.CheckNoServerError(t, err)
			}()

			httpClient := http.Client{
				Transport: &http.Transport{
					TLSClientConfig: certs.ClientTLSConf,
				},
			}

			time.Sleep(300 * time.Millisecond)
			uri, err := url.Parse("https://" + uncorsServer.Addr)
			testutils.CheckNoError(t, err)

			response, err := httpClient.Do(&http.Request{URL: uri, Method: http.MethodGet})
			testutils.CheckNoError(t, err)
			defer func() {
				testutils.CheckNoError(t, response.Body.Close())
			}()

			actualResponse, err := io.ReadAll(response.Body)
			testutils.CheckNoError(t, err)

			assert.Equal(t, expectedResponse, string(actualResponse))
		}))
	})

	t.Run("run already stopped server", func(t *testing.T) {
		uncorsServer := server.NewUncorsServer(ctx, handler)
		err := uncorsServer.Close()
		testutils.CheckNoServerError(t, err)

		t.Run("HTTP", func(t *testing.T) {
			err := uncorsServer.ListenAndServe("127.0.0.1:0")

			assert.ErrorIs(t, err, http.ErrServerClosed)
		})
		t.Run("HTTPS", testutils.WithTmpCerts(func(t *testing.T, certs *testutils.Certs) {
			err := uncorsServer.ListenAndServeTLS("127.0.0.1:0", certs.CertPath, certs.KeyPath)

			assert.ErrorIs(t, err, http.ErrServerClosed)
		}))
	})
}
