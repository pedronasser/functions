package tsuru

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pedronasser/runner/drivers"
	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
)

func NewTsuru(conf drivers.Config) (drivers.Driver, error) {
	return &tsuruDriver{
		conf: conf,
	}, nil
}

type tsuruDriver struct {
	conf   drivers.Config
	target string
}

func (d *tsuruDriver) Prepare(ctx context.Context, task drivers.ContainerTask) (drivers.Cookie, error) {
	return &tsuruCookie{
		drv:  d,
		task: task,
	}, nil
}

func (d *tsuruDriver) run(id string, task drivers.ContainerTask) (drivers.RunResult, error) {
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/run", task.Image()))
	if err != nil {
		return nil, err
	}
	v := url.Values{}
	payload, err := ioutil.ReadAll(task.Input())
	if err != nil {
		return nil, err
	}
	// Using base64 to send payload until tsuru supports attach to stdin
	v.Set("command", base64.StdEncoding.EncodeToString(payload))
	v.Set("once", "true")
	v.Set("isolated", "true")
	b := strings.NewReader(v.Encode())
	request, err := http.NewRequest("POST", u, b)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	_, stdout := task.Logger()
	w := tsuruIo.NewStreamWriter(stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, r.Body) {
	}
	if err != nil {
		return nil, err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return nil, fmt.Errorf("unparsed message error: %s", string(unparsed))
	}
	return nil, nil
}

func (d *tsuruDriver) stopApplication(id string) error {
	return nil
}

type tsuruCookie struct {
	id   string
	task drivers.ContainerTask
	drv  *tsuruDriver
}

func (c *tsuruCookie) Run(ctx context.Context) (drivers.RunResult, error) {
	return c.drv.run(c.id, c.task)
}

func (c *tsuruCookie) Close() error { return c.drv.stopApplication(c.id) }
