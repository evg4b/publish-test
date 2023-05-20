package handler_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-http-utils/headers"

	"github.com/evg4b/uncors/internal/helpers"

	"github.com/evg4b/uncors/internal/configuration"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

var (
	mock1Body = `{"mock": "mock number 1"}`
	mock2Body = `{"mock": "mock number 2"}`
	mock3Body = `{"mock": "mock number 3"}`
	mock4Body = `{"mock": "mock number 4"}`

	localhost       = "http://localhost"
	localhostSecure = "https://localhost"

	backgroundPng = "background.png"
	iconsSvg      = "icons.svg"
	indexJS       = "index.js"
	styleCSS      = "styles.css"
	indexHTML     = "index.html"
	mockJSON      = "mock.json"

	api     = "http://localhost/api"
	apiUser = "https://localhost/api/user"

	userPath = "/api/user"

	userIDHeader = "User-Id"
)

func TestUncorsRequestHandler(t *testing.T) {
	fs := testutils.FsFromMap(t, map[string]string{
		"/images/background.png": backgroundPng,
		"/images/svg/icons.svg":  iconsSvg,
		"/assets/js/index.js":    indexJS,
		"/assets/css/styles.css": styleCSS,
		"/assets/index.html":     indexHTML,
		"/mock.json":             mockJSON,
	})

	mappings := []configuration.URLMapping{
		{
			From: localhost,
			To:   localhostSecure,
			Statics: []configuration.StaticDirMapping{
				{Dir: "/assets", Path: "/cc/", Index: indexHTML},
				{Dir: "/assets", Path: "/pnp/", Index: "index.php"},
				{Dir: "/images", Path: "/img/"},
			},
		},
	}

	mockDefs := []configuration.Mock{
		{
			Path: "/api/mocks/1",
			Response: configuration.Response{
				Code:       http.StatusOK,
				RawContent: "mock-1",
			},
		},
		{
			Path: "/api/mocks/2",
			Response: configuration.Response{
				Code: http.StatusOK,
				File: "/mock.json",
			},
		},
		{
			Path: "/api/mocks/3",
			Response: configuration.Response{
				Code:       http.StatusMultiStatus,
				RawContent: "mock-3",
			},
		},
		{
			Path: "/api/mocks/4",
			Response: configuration.Response{
				Code: http.StatusOK,
				File: "/unknown.json",
			},
		},
	}

	factory, err := urlreplacer.NewURLReplacerFactory(mappings)
	testutils.CheckNoError(t, err)

	httpResponseMapping := map[string]string{
		"/img/original.png": "original.png",
	}

	httpMock := mocks.NewHTTPClientMock(t).DoMock.Set(func(request *http.Request) (*http.Response, error) {
		if response, ok := httpResponseMapping[request.URL.Path]; ok {
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(response)),
				Status:     http.StatusText(http.StatusOK),
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Request:    request,
			}, nil
		}

		panic(fmt.Sprintf("incorrect request: %s", request.URL.Path))
	})

	hand := handler.NewUncorsRequestHandler(
		handler.WithLogger(mocks.NewLoggerMock(t)),
		handler.WithMocks(mockDefs),
		handler.WithFileSystem(fs),
		handler.WithURLReplacerFactory(factory),
		handler.WithHTTPClient(httpMock),
		handler.WithMappings(mappings),
	)

	t.Run("statics directory", func(t *testing.T) {
		t.Run("with index file", func(t *testing.T) {
			t.Run("should return static file", func(t *testing.T) {
				tests := []struct {
					name     string
					url      string
					expected string
				}{
					{
						name:     indexHTML,
						url:      "http://localhost/cc/index.html",
						expected: indexHTML,
					},
					{
						name:     indexJS,
						url:      "http://localhost/cc/js/index.js",
						expected: indexJS,
					},
					{
						name:     styleCSS,
						url:      "http://localhost/cc/css/styles.css",
						expected: styleCSS,
					},
				}
				for _, testCase := range tests {
					t.Run(testCase.name, func(t *testing.T) {
						recorder := httptest.NewRecorder()
						request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
						helpers.NormaliseRequest(request)

						hand.ServeHTTP(recorder, request)

						assert.Equal(t, 200, recorder.Code)
						assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
					})
				}
			})

			t.Run("should return index file by default", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "http://localhost/cc/unknown.html", nil)
				helpers.NormaliseRequest(request)

				hand.ServeHTTP(recorder, request)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, indexHTML, testutils.ReadBody(t, recorder))
			})

			t.Run("should return error code when index file doesn't exists", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "http://localhost/pnp/unknown.html", nil)
				helpers.NormaliseRequest(request)

				hand.ServeHTTP(recorder, request)

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Contains(t, testutils.ReadBody(t, recorder), "Internal Server Error")
			})
		})

		t.Run("without index file", func(t *testing.T) {
			t.Run("should return static file", func(t *testing.T) {
				tests := []struct {
					name     string
					url      string
					expected string
				}{
					{
						name:     backgroundPng,
						url:      "http://localhost/img/background.png",
						expected: backgroundPng,
					},
					{
						name:     iconsSvg,
						url:      "http://localhost/img/svg/icons.svg",
						expected: iconsSvg,
					},
				}
				for _, testCase := range tests {
					t.Run(testCase.name, func(t *testing.T) {
						recorder := httptest.NewRecorder()
						request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
						helpers.NormaliseRequest(request)

						hand.ServeHTTP(recorder, request)

						assert.Equal(t, http.StatusOK, recorder.Code)
						assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
					})
				}
			})

			t.Run("should return original file", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "http://localhost/img/original.png", nil)
				helpers.NormaliseRequest(request)

				hand.ServeHTTP(recorder, request)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, "original.png", testutils.ReadBody(t, recorder))
			})
		})
	})

	t.Run("mocks", func(t *testing.T) {
		t.Run("should return mock with", func(t *testing.T) {
			tests := []struct {
				name         string
				url          string
				expected     string
				expectedCode int
			}{
				{
					name:         "raw content mock",
					url:          "http://localhost/api/mocks/1",
					expected:     "mock-1",
					expectedCode: http.StatusOK,
				},
				{
					name:         "file content mock",
					url:          "http://localhost/api/mocks/2",
					expected:     mockJSON,
					expectedCode: http.StatusOK,
				},
				{
					name:         "MultiStatus mock",
					url:          "http://localhost/api/mocks/3",
					expected:     "mock-3",
					expectedCode: http.StatusMultiStatus,
				},
			}
			for _, testCase := range tests {
				t.Run(testCase.name, func(t *testing.T) {
					recorder := httptest.NewRecorder()
					request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
					helpers.NormaliseRequest(request)

					hand.ServeHTTP(recorder, request)

					assert.Equal(t, testCase.expectedCode, recorder.Code)
					assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
				})
			}
		})

		t.Run("should return error code when mock file doesn't exists", func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "http://localhost/api/mocks/4", nil)
			helpers.NormaliseRequest(request)

			hand.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			assert.Contains(t, testutils.ReadBody(t, recorder), "Internal Server Error")
		})
	})
}

func TestMockMiddleware(t *testing.T) {
	logger := mocks.NewNoopLogger(t)

	t.Run("request method handling", func(t *testing.T) {
		t.Run("where mock method is not set allow method", func(t *testing.T) {
			middleware := handler.NewUncorsRequestHandler(
				handler.WithHTTPClient(mocks.NewHTTPClientMock(t)),
				handler.WithURLReplacerFactory(mocks.NewURLReplacerFactoryMock(t)),
				handler.WithLogger(logger),
				handler.WithMocks([]configuration.Mock{{
					Path: "/api",
					Response: configuration.Response{
						Code:       http.StatusOK,
						RawContent: mock1Body,
					},
				}}),
			)

			methods := []string{
				http.MethodGet,
				http.MethodHead,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodTrace,
			}

			for _, method := range methods {
				t.Run(method, func(t *testing.T) {
					request := httptest.NewRequest(method, api, nil)
					recorder := httptest.NewRecorder()

					middleware.ServeHTTP(recorder, request)

					body := testutils.ReadBody(t, recorder)
					assert.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, mock1Body, body)
				})
			}
		})

		t.Run("where method is set", func(t *testing.T) {
			expectedCode := 299
			expectedBody := "forwarded"
			factory, err := urlreplacer.NewURLReplacerFactory([]configuration.URLMapping{
				{From: "*", To: "*"},
			})
			testutils.CheckNoError(t, err)

			middleware := handler.NewUncorsRequestHandler(
				handler.WithHTTPClient(mocks.NewHTTPClientMock(t).DoMock.
					Set(func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							Request:    req,
							StatusCode: expectedCode,
							Body:       io.NopCloser(strings.NewReader(expectedBody)),
						}, nil
					})),
				handler.WithURLReplacerFactory(factory),
				handler.WithLogger(logger),
				handler.WithMocks([]configuration.Mock{{
					Path:   "/api",
					Method: http.MethodPut,
					Response: configuration.Response{
						Code:       http.StatusOK,
						RawContent: mock1Body,
					},
				}}),
			)

			t.Run("method is not matched", func(t *testing.T) {
				methods := []string{
					http.MethodGet,
					http.MethodHead,
					http.MethodPost,
					http.MethodPatch,
					http.MethodDelete,
					http.MethodTrace,
				}

				for _, method := range methods {
					t.Run(method, func(t *testing.T) {
						request := httptest.NewRequest(method, api, nil)
						recorder := httptest.NewRecorder()

						middleware.ServeHTTP(recorder, request)

						assert.Equal(t, expectedCode, recorder.Code)
						assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
					})
				}

				t.Run(http.MethodOptions, func(t *testing.T) {
					request := httptest.NewRequest(http.MethodOptions, api, nil)
					recorder := httptest.NewRecorder()

					middleware.ServeHTTP(recorder, request)

					assert.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, "", testutils.ReadBody(t, recorder))
				})
			})

			t.Run("method is matched", func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPut, api, nil)
				recorder := httptest.NewRecorder()

				middleware.ServeHTTP(recorder, request)

				body := testutils.ReadBody(t, recorder)
				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, mock1Body, body)
			})
		})
	})

	t.Run("path handling", func(t *testing.T) {
		expectedCode := 299
		expectedBody := "forwarded"
		factory, err := urlreplacer.NewURLReplacerFactory([]configuration.URLMapping{
			{From: "*", To: "*"},
		})
		testutils.CheckNoError(t, err)

		middleware := handler.NewUncorsRequestHandler(
			handler.WithHTTPClient(mocks.NewHTTPClientMock(t).DoMock.
				Set(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						Request:    req,
						StatusCode: expectedCode,
						Body:       io.NopCloser(strings.NewReader(expectedBody)),
					}, nil
				})),
			handler.WithURLReplacerFactory(factory),
			handler.WithLogger(logger),
			handler.WithMocks([]configuration.Mock{
				{
					Path: userPath,
					Response: configuration.Response{
						Code:       http.StatusOK,
						RawContent: mock1Body,
					},
				},
				{
					Path: "/api/user/{id:[0-9]+}",
					Response: configuration.Response{
						Code:       http.StatusAccepted,
						RawContent: mock2Body,
					},
				},
				{
					Path: "/api/{single-path/demo",
					Response: configuration.Response{
						Code:       http.StatusBadRequest,
						RawContent: mock3Body,
					},
				},
				{
					Path: `/api/v2/{multiple-path:[a-z-\/]+}/demo`,
					Response: configuration.Response{
						Code:       http.StatusCreated,
						RawContent: mock4Body,
					},
				},
			}),
		)

		tests := []struct {
			name       string
			url        string
			expected   string
			statusCode int
		}{
			{
				name:       "direct path",
				url:        apiUser,
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name:       "direct path with ending slash",
				url:        "https://localhost/api/user/",
				expected:   expectedBody,
				statusCode: expectedCode,
			},
			{
				name:       "direct path with parameter",
				url:        "https://localhost/api/user/23",
				expected:   mock2Body,
				statusCode: http.StatusAccepted,
			},
			{
				name:       "direct path with incorrect parameter",
				url:        "https://localhost/api/user/unknow",
				expected:   expectedBody,
				statusCode: expectedCode,
			},
			{
				name:       "path with subpath to single matching param",
				url:        "https://localhost/api/some-path/with-some-subpath/demo",
				expected:   expectedBody,
				statusCode: expectedCode,
			},
			{
				name:       "path with subpath to multiple matching param",
				url:        "https://localhost/api/v2/some-path/with-some-subpath/demo",
				expected:   mock4Body,
				statusCode: http.StatusCreated,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
				recorder := httptest.NewRecorder()

				middleware.ServeHTTP(recorder, request)

				body := testutils.ReadBody(t, recorder)
				assert.Equal(t, testCase.statusCode, recorder.Code)
				assert.Equal(t, testCase.expected, body)
			})
		}
	})

	t.Run("query handling", func(t *testing.T) {
		middleware := handler.NewUncorsRequestHandler(
			handler.WithHTTPClient(mocks.NewHTTPClientMock(t)),
			handler.WithURLReplacerFactory(mocks.NewURLReplacerFactoryMock(t)),
			handler.WithLogger(logger),
			handler.WithMocks([]configuration.Mock{
				{
					Path: userPath,
					Response: configuration.Response{
						Code:       http.StatusOK,
						RawContent: mock1Body,
					},
				},
				{
					Path: userPath,
					Queries: map[string]string{
						"id": "17",
					},
					Response: configuration.Response{
						Code:       http.StatusCreated,
						RawContent: mock2Body,
					},
				},
				{
					Path: userPath,
					Queries: map[string]string{
						"id":    "99",
						"token": "fe145b54563d9be1b2a476f56b0a412b",
					},
					Response: configuration.Response{
						Code:       http.StatusAccepted,
						RawContent: mock3Body,
					},
				},
			}),
		)

		tests := []struct {
			name       string
			url        string
			expected   string
			statusCode int
		}{
			{
				name:       "queries is not set",
				url:        "http://localhost/api/user",
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name:       "passed unsetted parameter",
				url:        "http://localhost/api/user?id=16",
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name:       "passed parameter",
				url:        "http://localhost/api/user?id=17",
				expected:   mock2Body,
				statusCode: http.StatusCreated,
			},
			{
				name:       "passed one of multiple parameters",
				url:        "http://localhost/api/user?id=99",
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name:       "passed all of multiple parameters",
				url:        "http://localhost/api/user?id=99&token=fe145b54563d9be1b2a476f56b0a412b",
				expected:   mock3Body,
				statusCode: http.StatusAccepted,
			},
			{
				name:       "passed extra parameters",
				url:        "http://localhost/api/user?id=99&token=fe145b54563d9be1b2a476f56b0a412b&demo=true",
				expected:   mock3Body,
				statusCode: http.StatusAccepted,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
				recorder := httptest.NewRecorder()

				middleware.ServeHTTP(recorder, request)

				body := testutils.ReadBody(t, recorder)
				assert.Equal(t, testCase.statusCode, recorder.Code)
				assert.Equal(t, testCase.expected, body)
			})
		}
	})

	t.Run("header handling", func(t *testing.T) {
		middleware := handler.NewUncorsRequestHandler(
			handler.WithHTTPClient(mocks.NewHTTPClientMock(t)),
			handler.WithURLReplacerFactory(mocks.NewURLReplacerFactoryMock(t)),
			handler.WithLogger(logger),
			handler.WithMocks([]configuration.Mock{
				{
					Path: userPath,
					Response: configuration.Response{
						Code:       http.StatusOK,
						RawContent: mock1Body,
					},
				},
				{
					Path: userPath,
					Headers: map[string]string{
						headers.XCSRFToken: "de4e27987d054577b0edc0e828851724",
					},
					Response: configuration.Response{
						Code:       http.StatusCreated,
						RawContent: mock2Body,
					},
				},
				{
					Path: userPath,
					Headers: map[string]string{
						userIDHeader:       "99",
						headers.XCSRFToken: "fe145b54563d9be1b2a476f56b0a412b",
					},
					Response: configuration.Response{
						Code:       http.StatusAccepted,
						RawContent: mock3Body,
					},
				},
			}),
		)

		tests := []struct {
			name       string
			url        string
			headers    map[string]string
			expected   string
			statusCode int
		}{
			{
				name:       "headers is not set",
				url:        apiUser,
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name: "passed unsetted headers",
				url:  apiUser,
				headers: map[string]string{
					headers.XCSRFToken: "55cc413b96026e833835a2c9a3f39c21",
				},
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name: "passed defined header",
				url:  apiUser,
				headers: map[string]string{
					headers.XCSRFToken: "de4e27987d054577b0edc0e828851724",
				},
				expected:   mock2Body,
				statusCode: http.StatusCreated,
			},
			{
				name: "passed one of multiple headers",
				url:  apiUser,
				headers: map[string]string{
					userIDHeader: "99",
				},
				expected:   mock1Body,
				statusCode: http.StatusOK,
			},
			{
				name: "passed all of multiple headers",
				url:  apiUser,
				headers: map[string]string{
					userIDHeader:       "99",
					headers.XCSRFToken: "fe145b54563d9be1b2a476f56b0a412b",
				},
				expected:   mock3Body,
				statusCode: http.StatusAccepted,
			},
			{
				name: "passed extra headers",
				url:  apiUser,
				headers: map[string]string{
					userIDHeader:           "99",
					headers.XCSRFToken:     "fe145b54563d9be1b2a476f56b0a412b",
					headers.AcceptEncoding: "deflate",
				},
				expected:   mock3Body,
				statusCode: http.StatusAccepted,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPost, testCase.url, nil)
				for key, value := range testCase.headers {
					request.Header.Add(key, value)
				}
				recorder := httptest.NewRecorder()

				middleware.ServeHTTP(recorder, request)

				body := testutils.ReadBody(t, recorder)
				assert.Equal(t, testCase.statusCode, recorder.Code)
				assert.Equal(t, testCase.expected, body)
			})
		}
	})
}