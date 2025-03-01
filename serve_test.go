package fuego

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type ans struct {
	Ans string `json:"ans"`
}

func testController(c *ContextNoBody) (ans, error) {
	return ans{Ans: "Hello World"}, nil
}

func testControllerWithError(c *ContextNoBody) (ans, error) {
	return ans{}, errors.New("error happened!")
}

type testOutTransformer struct {
	Name     string `json:"name"`
	Password string `json:"ans"`
}

func (t *testOutTransformer) OutTransform(ctx context.Context) error {
	t.Name = "M. " + t.Name
	t.Password = "redacted"
	return nil
}

var _ OutTransformer = &testOutTransformer{}

func testControllerWithOutTransformer(c *ContextNoBody) (testOutTransformer, error) {
	return testOutTransformer{Name: "John"}, nil
}

func testControllerWithOutTransformerStar(c *ContextNoBody) (*testOutTransformer, error) {
	return &testOutTransformer{Name: "John"}, nil
}

func testControllerWithOutTransformerStarError(c *ContextNoBody) (*testOutTransformer, error) {
	return nil, errors.New("error happened!")
}

func testControllerWithOutTransformerStarNil(c *ContextNoBody) (*testOutTransformer, error) {
	return nil, nil
}

func testControllerReturningString(c *ContextNoBody) (string, error) {
	return "hello world", nil
}

func testControllerReturningPtrToString(c *ContextNoBody) (*string, error) {
	s := "hello world"
	return &s, nil
}

func TestHttpHandler(t *testing.T) {
	s := NewServer()

	t.Run("can create std http handler from fuego controller", func(t *testing.T) {
		handler := httpHandler[ans, any](s, testController)
		if handler == nil {
			t.Error("handler is nil")
		}
	})

	t.Run("can run http handler from fuego controller", func(t *testing.T) {
		handler := httpHandler(s, testController)

		req := httptest.NewRequest("GET", "/testing", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		body := w.Body.String()
		require.Equal(t, crlf(`{"ans":"Hello World"}`), body)
	})

	t.Run("can handle errors in http handler from fuego controller", func(t *testing.T) {
		handler := httpHandler(s, testControllerWithError)
		if handler == nil {
			t.Error("handler is nil")
		}

		req := httptest.NewRequest("GET", "/testing", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		body := w.Body.String()
		require.Equal(t, crlf(`{"error":"error happened!"}`), body)
	})

	t.Run("can outTransform before serializing a value", func(t *testing.T) {
		handler := httpHandler(s, testControllerWithOutTransformer)

		req := httptest.NewRequest("GET", "/testing", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		body := w.Body.String()
		require.Equal(t, crlf(`{"name":"M. John","ans":"redacted"}`), body)
	})

	t.Run("can outTransform before serializing a pointer value", func(t *testing.T) {
		handler := httpHandler(s, testControllerWithOutTransformerStar)

		req := httptest.NewRequest("GET", "/testing", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		body := w.Body.String()
		require.Equal(t, crlf(`{"name":"M. John","ans":"redacted"}`), body)
	})

	t.Run("can handle errors in outTransform", func(t *testing.T) {
		handler := httpHandler(s, testControllerWithOutTransformerStarError)

		req := httptest.NewRequest("GET", "/testing", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		body := w.Body.String()
		require.Equal(t, crlf(`{"error":"error happened!"}`), body)
	})

	t.Run("can handle nil in outTransform", func(t *testing.T) {
		handler := httpHandler(s, testControllerWithOutTransformerStarNil)

		req := httptest.NewRequest("GET", "/testing", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		body := w.Body.String()
		require.Equal(t, "null\n", body)
	})

	t.Run("returns correct content-type when returning string", func(t *testing.T) {
		handler := httpHandler(s, testControllerReturningString)

		req := httptest.NewRequest("GET", "/testing", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		require.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	})

	t.Run("returns correct content-type when returning ptr to string", func(t *testing.T) {
		handler := httpHandler(s, testControllerReturningPtrToString)

		req := httptest.NewRequest("GET", "/testing", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		require.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	})
}

func TestServer_Run(t *testing.T) {
	// This is not a standard test, it is here to ensure that the server can run.
	// Please do not run this kind of test for your controllers, it is NOT unit testing.
	t.Run("can run server", func(t *testing.T) {
		s := NewServer(
			WithoutLogger(),
		)

		Get(s, "/test", func(ctx *ContextNoBody) (string, error) {
			return "OK", nil
		})

		go func() {
			s.Run()
		}()

		require.Eventually(t, func() bool {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			s.Mux.ServeHTTP(w, req)

			return w.Body.String() == `OK`
		}, 5*time.Millisecond, 500*time.Microsecond)
	})
}

func TestSetStatusBeforeSend(t *testing.T) {
	s := NewServer()

	t.Run("can set status before sending", func(t *testing.T) {
		handler := httpHandler(s, func(c *ContextNoBody) (ans, error) {
			c.Response().WriteHeader(201)
			return ans{Ans: "Hello World"}, nil
		})

		req := httptest.NewRequest("GET", "/testing", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		require.Equal(t, 201, w.Code)

		body := w.Body.String()
		require.Equal(t, crlf(`{"ans":"Hello World"}`), body)
	})

	t.Run("can set status with the shortcut before sending", func(t *testing.T) {
		handler := httpHandler(s, func(c *ContextNoBody) (ans, error) {
			c.SetStatus(202)
			return ans{Ans: "Hello World"}, nil
		})

		req := httptest.NewRequest("GET", "/testing", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		require.Equal(t, 202, w.Code)

		body := w.Body.String()
		require.Equal(t, crlf(`{"ans":"Hello World"}`), body)
	})
}

type testRenderer struct{}

func (t testRenderer) Render(w io.Writer) error {
	w.Write([]byte("hello"))
	return nil
}

type testCtxRenderer struct{}

func (t testCtxRenderer) Render(ctx context.Context, w io.Writer) error {
	w.Write([]byte("world"))
	return nil
}

type testErrorRenderer struct{}

func (t testErrorRenderer) Render(w io.Writer) error { return errors.New("cannot render") }

type testCtxErrorRenderer struct{}

func (t testCtxErrorRenderer) Render(ctx context.Context, w io.Writer) error {
	return errors.New("cannot render")
}

func TestServeRenderer(t *testing.T) {
	s := NewServer(
		WithErrorSerializer(func(w http.ResponseWriter, err error) {
			w.WriteHeader(500)
			w.Write([]byte("<body><h1>error</h1></body>"))
		}),
	)

	t.Run("can serve renderer", func(t *testing.T) {
		Get(s, "/", func(c *ContextNoBody) (Renderer, error) {
			return testRenderer{}, nil
		})
		Get(s, "/error-in-controller", func(c *ContextNoBody) (Renderer, error) {
			return nil, errors.New("error")
		})
		Get(s, "/error-in-rendering", func(c *ContextNoBody) (Renderer, error) {
			return testErrorRenderer{}, nil
		})

		t.Run("normal return", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			s.Mux.ServeHTTP(w, req)

			require.Equal(t, 200, w.Code)
			require.Equal(t, "hello", w.Body.String())
		})

		t.Run("error return", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/error-in-controller", nil)
			w := httptest.NewRecorder()
			s.Mux.ServeHTTP(w, req)

			require.Equal(t, 500, w.Code)
			require.Equal(t, "<body><h1>error</h1></body>", w.Body.String())
		})

		t.Run("error in rendering", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/error-in-rendering", nil)
			w := httptest.NewRecorder()
			s.Mux.ServeHTTP(w, req)

			require.Equal(t, 500, w.Code)
			require.Equal(t, "<body><h1>error</h1></body>", w.Body.String())
		})
	})

	t.Run("can serve ctx renderer", func(t *testing.T) {
		Get(s, "/ctx", func(c *ContextNoBody) (CtxRenderer, error) {
			return testCtxRenderer{}, nil
		})
		Get(s, "/ctx/error-in-controller", func(c *ContextNoBody) (CtxRenderer, error) {
			return nil, errors.New("error")
		})
		Get(s, "/ctx/error-in-rendering", func(c *ContextNoBody) (CtxRenderer, error) {
			return testCtxErrorRenderer{}, nil
		})

		t.Run("normal return", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ctx", nil)
			w := httptest.NewRecorder()
			s.Mux.ServeHTTP(w, req)

			require.Equal(t, 200, w.Code)
			require.Equal(t, "world", w.Body.String())
		})

		t.Run("error return", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ctx/error-in-controller", nil)
			w := httptest.NewRecorder()
			s.Mux.ServeHTTP(w, req)

			require.Equal(t, 500, w.Code)
			require.Equal(t, "<body><h1>error</h1></body>", w.Body.String())
		})

		t.Run("error in rendering", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ctx/error-in-rendering", nil)
			w := httptest.NewRecorder()
			s.Mux.ServeHTTP(w, req)

			require.Equal(t, 500, w.Code)
			require.Equal(t, "<body><h1>error</h1></body>", w.Body.String())
		})
	})
}

func TestIni(t *testing.T) {
	t.Run("can initialize ContextNoBody", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ctx/error-in-rendering", nil)
		w := httptest.NewRecorder()
		ctx := initContext[ContextNoBody](ContextNoBody{
			request:  req,
			response: w,
		})

		require.NotNil(t, ctx)
		require.NotNil(t, ctx.Request())
		require.NotNil(t, ctx.Response())
	})

	t.Run("can initialize ContextNoBody", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ctx/error-in-rendering", nil)
		w := httptest.NewRecorder()
		ctx := initContext[*ContextNoBody](ContextNoBody{
			request:  req,
			response: w,
		})

		require.NotNil(t, ctx)
		require.NotNil(t, ctx.Request())
		require.NotNil(t, ctx.Response())
	})

	t.Run("can initialize ContextWithBody[string]", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ctx/error-in-rendering", nil)
		w := httptest.NewRecorder()
		ctx := initContext[*ContextWithBody[string]](ContextNoBody{
			request:  req,
			response: w,
		})

		require.NotNil(t, ctx)
		require.NotNil(t, ctx.Request())
		require.NotNil(t, ctx.Response())
	})

	t.Run("can initialize ContextWithBody[struct]", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ctx/error-in-rendering", nil)
		w := httptest.NewRecorder()
		ctx := initContext[*ContextWithBody[ans]](ContextNoBody{
			request:  req,
			response: w,
		})

		require.NotNil(t, ctx)
		require.NotNil(t, ctx.Request())
		require.NotNil(t, ctx.Response())
	})

	t.Run("cannot initialize with Ctx interface", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ctx/error-in-rendering", nil)
		w := httptest.NewRecorder()

		require.Panics(t, func() {
			initContext[ctx[any]](ContextNoBody{
				request:  req,
				response: w,
			})
		})
	})
}
