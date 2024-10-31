package main

import (
	"errors"
)

func TestSniffTest(td *TD) {
	xr := td.Props["xr"].(*XRegistry)
	td.Log("Server URL: %s", xr.GetServerURL())
	res := xr.Get("")
	if res.Error != nil || res.StatusCode != 200 {
		td.Fail("'GET /' MUST return 200, not %d(%s)", res.StatusCode,
			errors.Unwrap(res.Error))
	}

	td.Must(len(res.Body) > 0, "'GET /' MUST return a non-empty body")

	if res.JSON == nil {
		tmp := " <empty>"
		if len(res.Body) > 0 {
			tmp = "\n" + string(res.Body)
		}
		td.Fail("'GET /' MUST return a JSON body, not:%s", tmp)
	}

	td.Log("GET / returned 200 + JSON body")
}

func TestLoadModel(td *TD) {
	td.DependsOn(TestSniffTest)
	xr := td.Props["xr"].(*XRegistry)

	res := xr.Get("/model")
	td.Must(res.StatusCode == 200,
		"'GET /model' MUST return the model, not %d(%s)",
		res.StatusCode, errors.Unwrap(res.Error))

	td.Must(res.JSON != nil, "The model MUST NOT be empty")

	Model = res.JSON
	td.Log("Model:\n%s", ToJSON(Model))
}
