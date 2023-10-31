package op

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type ans struct {
	Ans string `json:"ans"`
}

func testController(c Ctx[any]) (ans, error) {
	return ans{Ans: "Hello World"}, nil
}

func testControllerWithError(c Ctx[any]) (ans, error) {
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

func testControllerWithOutTransformer(c Ctx[any]) (testOutTransformer, error) {
	return testOutTransformer{Name: "John"}, nil
}

func testControllerWithOutTransformerStar(c Ctx[any]) (*testOutTransformer, error) {
	return &testOutTransformer{Name: "John"}, nil
}

func testControllerWithOutTransformerStarError(c Ctx[any]) (*testOutTransformer, error) {
	return nil, errors.New("error happened!")
}

func testControllerWithOutTransformerStarNil(c Ctx[any]) (*testOutTransformer, error) {
	return nil, nil
}

func TestHttpHandler(t *testing.T) {
	s := NewServer()

	t.Run("can create std http handler from op controller", func(t *testing.T) {
		handler := httpHandler[ans, any](s, testController)
		if handler == nil {
			t.Error("handler is nil")
		}
	})

	t.Run("can run http handler from op controller", func(t *testing.T) {
		handler := httpHandler(s, testController)

		req := httptest.NewRequest("GET", "/testing", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		body := w.Body.String()
		require.Equal(t, crlf(`{"ans":"Hello World"}`), body)
	})

	t.Run("can handle errors in http handler from op controller", func(t *testing.T) {
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
}
