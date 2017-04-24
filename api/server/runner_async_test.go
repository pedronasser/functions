package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pedronasser/functions/api/datastore"
	"github.com/pedronasser/functions/api/models"
	"github.com/pedronasser/functions/api/mqs"
	"github.com/pedronasser/functions/api/runner"
	"github.com/pedronasser/functions/api/runner/task"
	"github.com/pedronasser/functions/api/server/internal/routecache"
)

func testRouterAsync(ds models.Datastore, mq models.MessageQueue, rnr *runner.Runner, tasks chan task.Request, enqueue models.Enqueue) *gin.Engine {
	ctx := context.Background()

	s := &Server{
		Runner:    rnr,
		Router:    gin.New(),
		Datastore: ds,
		MQ:        mq,
		tasks:     tasks,
		Enqueue:   enqueue,
		hotroutes: routecache.New(10),
	}

	r := s.Router
	r.Use(gin.Logger())

	r.Use(prepareMiddleware(ctx))
	s.bindHandlers(ctx)
	return r
}

func TestRouteRunnerAsyncExecution(t *testing.T) {
	tasks := mockTasksConduit()
	ds := datastore.NewMockInit(
		[]*models.App{
			{Name: "myapp", Config: map[string]string{"app": "true"}},
		},
		[]*models.Route{
			{Type: "async", Path: "/myroute", AppName: "myapp", Image: "iron/hello", Config: map[string]string{"test": "true"}},
			{Type: "async", Path: "/myerror", AppName: "myapp", Image: "iron/error", Config: map[string]string{"test": "true"}},
			{Type: "async", Path: "/myroute/:param", AppName: "myapp", Image: "iron/hello", Config: map[string]string{"test": "true"}},
		},
	)
	mq := &mqs.Mock{}

	for i, test := range []struct {
		path         string
		body         string
		headers      map[string][]string
		expectedCode int
		expectedEnv  map[string]string
	}{
		{"/r/myapp/myroute", ``, map[string][]string{}, http.StatusAccepted, map[string]string{"TEST": "true", "APP": "true"}},
		// FIXME: this just hangs
		//{"/r/myapp/myroute/1", ``, map[string][]string{}, http.StatusAccepted, map[string]string{"TEST": "true", "APP": "true"}},
		{"/r/myapp/myerror", ``, map[string][]string{}, http.StatusAccepted, map[string]string{"TEST": "true", "APP": "true"}},
		{"/r/myapp/myroute", `{ "name": "test" }`, map[string][]string{}, http.StatusAccepted, map[string]string{"TEST": "true", "APP": "true"}},
		{
			"/r/myapp/myroute",
			``,
			map[string][]string{"X-Function": []string{"test"}},
			http.StatusAccepted,
			map[string]string{
				"TEST":              "true",
				"APP":               "true",
				"HEADER_X_FUNCTION": "test",
			},
		},
	} {
		body := bytes.NewBuffer([]byte(test.body))
		var wg sync.WaitGroup

		wg.Add(1)
		fmt.Println("About to start router")
		rnr, cancel := testRunner(t)
		router := testRouterAsync(ds, mq, rnr, tasks, func(_ context.Context, _ models.MessageQueue, task *models.Task) (*models.Task, error) {
			if test.body != task.Payload {
				t.Errorf("Test %d: Expected task Payload to be the same as the test body", i)
			}

			if test.expectedEnv != nil {
				for name, value := range test.expectedEnv {
					taskName := name
					if value != task.EnvVars[taskName] {
						t.Errorf("Test %d: Expected header `%s` to be `%s` but was `%s`",
							i, name, value, task.EnvVars[taskName])
					}
				}
			}

			wg.Done()
			return task, nil
		})

		fmt.Println("makeing requests")
		req, rec := newRouterRequest(t, "POST", test.path, body)
		for name, value := range test.headers {
			req.Header.Set(name, value[0])
		}
		fmt.Println("About to start router2")
		router.ServeHTTP(rec, req)
		fmt.Println("after servehttp")

		if rec.Code != test.expectedCode {
			t.Errorf("Test %d: Expected status code to be %d but was %d",
				i, test.expectedCode, rec.Code)
		}

		wg.Wait()
		cancel()
	}
}
