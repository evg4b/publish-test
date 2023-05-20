package mock_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/handler/mock"

	"github.com/evg4b/uncors/internal/configuration"

	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

const (
	textPlain = "text/plain; charset=utf-8"
	imagePng  = "image/png"
)

const (
	textContent = "status: ok"
	jsonContent = `{ "test": "ok" }`
	htmlContent = "<html></html>"
	pngContent  = "\x89PNG\x0D\x0A\x1A\x0A"
)

const (
	textFile = "test.txt"
	jsonFile = "test.json"
	htmlFile = "test.html"
	pngFile  = "test.png"
)

func TestHandler(t *testing.T) {
	fileSystem := testutils.FsFromMap(t, map[string]string{
		textFile: textContent,
		jsonFile: jsonContent,
		htmlFile: htmlContent,
		pngFile:  pngContent,
	})

	t.Run("mock content", func(t *testing.T) {
		tests := []struct {
			name     string
			response configuration.Response
			expected string
		}{
			{
				name:     "raw content",
				response: configuration.Response{RawContent: jsonContent},
				expected: jsonContent,
			},
			{
				name:     "file content",
				response: configuration.Response{File: jsonFile},
				expected: jsonContent,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := mock.NewMockMiddleware(
					mock.WithLogger(mocks.NewNoopLogger(t)),
					mock.WithResponse(testCase.response),
					mock.WithFileSystem(fileSystem),
					mock.WithAfter(func(duration time.Duration) <-chan time.Time {
						return time.After(time.Nanosecond)
					}),
				)

				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "/", nil)
				handler.ServeHTTP(recorder, request)

				body := testutils.ReadBody(t, recorder)
				assert.EqualValues(t, testCase.expected, body)
			})
		}
	})

	t.Run("content type detection", func(t *testing.T) {
		tests := []struct {
			name     string
			response configuration.Response
			expected string
		}{
			{
				name:     "raw content with plain text",
				response: configuration.Response{RawContent: textContent},
				expected: textPlain,
			},
			{
				name:     "raw content with json",
				response: configuration.Response{RawContent: jsonContent},
				expected: textPlain,
			},
			{
				name:     "raw content with html",
				response: configuration.Response{RawContent: htmlContent},
				expected: "text/html; charset=utf-8",
			},
			{
				name:     "raw content with png",
				response: configuration.Response{RawContent: pngContent},
				expected: imagePng,
			},
			{
				name:     "file with plain text",
				response: configuration.Response{File: textFile},
				expected: textPlain,
			},
			{
				name:     "file with json",
				response: configuration.Response{File: jsonFile},
				expected: "application/json",
			},
			{
				name:     "file with html",
				response: configuration.Response{File: htmlFile},
				expected: "text/html; charset=utf-8",
			},
			{
				name:     "file with png",
				response: configuration.Response{File: pngFile},
				expected: imagePng,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := mock.NewMockMiddleware(
					mock.WithLogger(mocks.NewNoopLogger(t)),
					mock.WithResponse(testCase.response),
					mock.WithFileSystem(fileSystem),
					mock.WithAfter(func(duration time.Duration) <-chan time.Time {
						return time.After(time.Nanosecond)
					}),
				)

				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "/", nil)
				handler.ServeHTTP(recorder, request)

				header := testutils.ReadHeader(t, recorder)
				assert.EqualValues(t, testCase.expected, header.Get(headers.ContentType))
			})
		}
	})

	t.Run("headers settings", func(t *testing.T) {
		tests := []struct {
			name     string
			response configuration.Response
			expected http.Header
		}{
			{
				name: "should put default CORS headers",
				response: configuration.Response{
					Code:       http.StatusOK,
					RawContent: textContent,
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {"*"},
					headers.AccessControlAllowCredentials: {"true"},
					headers.ContentType:                   {textPlain},
					headers.AccessControlAllowMethods:     {mocks.AllMethods},
				},
			},
			{
				name: "should set response code",
				response: configuration.Response{
					Code:       http.StatusOK,
					RawContent: textContent,
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {"*"},
					headers.AccessControlAllowCredentials: {"true"},
					headers.ContentType:                   {textPlain},
					headers.AccessControlAllowMethods:     {mocks.AllMethods},
				},
			},
			{
				name: "should set custom headers",
				response: configuration.Response{
					Code: http.StatusOK,
					Headers: map[string]string{
						"X-Key": "X-Key-Value",
					},
					RawContent: textContent,
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {"*"},
					headers.AccessControlAllowCredentials: {"true"},
					headers.ContentType:                   {textPlain},
					"X-Key":                               {"X-Key-Value"},
					headers.AccessControlAllowMethods:     {mocks.AllMethods},
				},
			},
			{
				name: "should override default headers",
				response: configuration.Response{
					Code: http.StatusOK,
					Headers: map[string]string{
						headers.AccessControlAllowOrigin:      "localhost",
						headers.AccessControlAllowCredentials: "false",
						headers.ContentType:                   "none",
					},
					RawContent: textContent,
				},
				expected: map[string][]string{
					headers.AccessControlAllowOrigin:      {"localhost"},
					headers.AccessControlAllowCredentials: {"false"},
					headers.ContentType:                   {"none"},
					headers.AccessControlAllowMethods:     {mocks.AllMethods},
				},
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := mock.NewMockMiddleware(
					mock.WithLogger(mocks.NewNoopLogger(t)),
					mock.WithResponse(testCase.response),
					mock.WithFileSystem(fileSystem),
					mock.WithAfter(func(duration time.Duration) <-chan time.Time {
						return time.After(time.Nanosecond)
					}),
				)

				request := httptest.NewRequest(http.MethodGet, "/", nil)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)

				assert.EqualValues(t, testCase.expected, testutils.ReadHeader(t, recorder))
				assert.Equal(t, http.StatusOK, recorder.Code)
			})
		}
	})

	t.Run("status code", func(t *testing.T) {
		tests := []struct {
			name     string
			response configuration.Response
			expected int
		}{
			{
				name: "provide 201 code",
				response: configuration.Response{
					Code: http.StatusCreated,
				},
				expected: http.StatusCreated,
			},
			{
				name: "provide 503 code",
				response: configuration.Response{
					Code: http.StatusServiceUnavailable,
				},
				expected: http.StatusServiceUnavailable,
			},
			{
				name:     "automatically provide 200 code",
				response: configuration.Response{},
				expected: http.StatusOK,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := mock.NewMockMiddleware(
					mock.WithLogger(mocks.NewNoopLogger(t)),
					mock.WithResponse(testCase.response),
					mock.WithFileSystem(fileSystem),
					mock.WithAfter(func(duration time.Duration) <-chan time.Time {
						return time.After(time.Nanosecond)
					}),
				)

				request := httptest.NewRequest(http.MethodGet, "/", nil)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)

				assert.Equal(t, testCase.expected, recorder.Code)
			})
		}
	})

	t.Run("mock response delay", func(t *testing.T) {
		t.Run("correctly handle delay", func(t *testing.T) {
			tests := []struct {
				name           string
				response       configuration.Response
				shouldBeCalled bool
				expected       time.Duration
			}{
				{
					name: "3s delay",
					response: configuration.Response{
						Code:  http.StatusCreated,
						Delay: 3 * time.Second,
					},
					shouldBeCalled: true,
					expected:       3 * time.Second,
				},
				{
					name: "15h delay",
					response: configuration.Response{
						Code:  http.StatusCreated,
						Delay: 15 * time.Hour,
					},
					shouldBeCalled: true,
					expected:       15 * time.Hour,
				},
				{
					name: "0s delay",
					response: configuration.Response{
						Code:  http.StatusCreated,
						Delay: 0 * time.Second,
					},
					shouldBeCalled: false,
				},
				{
					name: "delay is not set",
					response: configuration.Response{
						Code: http.StatusCreated,
					},
					shouldBeCalled: false,
				},
				{
					name: "incorrect delay",
					response: configuration.Response{
						Code:  http.StatusCreated,
						Delay: -13 * time.Minute,
					},
					shouldBeCalled: false,
				},
			}
			for _, testCase := range tests {
				t.Run(testCase.name, func(t *testing.T) {
					called := false
					handler := mock.NewMockMiddleware(
						mock.WithLogger(mocks.NewNoopLogger(t)),
						mock.WithResponse(testCase.response),
						mock.WithFileSystem(fileSystem),
						mock.WithAfter(func(duration time.Duration) <-chan time.Time {
							assert.Equal(t, duration, testCase.expected)
							called = true

							return time.After(time.Nanosecond)
						}),
					)

					request := httptest.NewRequest(http.MethodGet, "/", nil)
					recorder := httptest.NewRecorder()

					handler.ServeHTTP(recorder, request)

					assert.Equal(t, called, testCase.shouldBeCalled)
				})
			}
		})

		t.Run("correctly cancel delay", func(t *testing.T) {
			handler := mock.NewMockMiddleware(
				mock.WithLogger(mocks.NewNoopLogger(t)),
				mock.WithResponse(configuration.Response{
					Code:       http.StatusOK,
					Delay:      1 * time.Hour,
					RawContent: "Text content",
				}),
				mock.WithFileSystem(fileSystem),
				mock.WithAfter(time.After),
			)

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			ctx, cancel := context.WithCancel(context.Background())
			recorder := httptest.NewRecorder()

			var waitGroup sync.WaitGroup
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				handler.ServeHTTP(recorder, request.WithContext(ctx))
			}()

			cancel()

			waitGroup.Wait()

			assert.Equal(t, testutils.ReadBody(t, recorder), "")
			assert.Equal(t, recorder.Code, http.StatusServiceUnavailable)
		})
	})
}