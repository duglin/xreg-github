package tests

import (
	"testing"
	// "github.com/duglin/xreg-github/registry"
)

func TestTypeMap(t *testing.T) {
	reg := NewRegistry("TestTypeMap")
	defer PassDeleteReg(t, reg)
	xCheck(t, reg != nil, "can't create reg")

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)

	xCheck(t, rm.TypeMap == nil, "Should be empty")

	xNoErr(t, rm.AddTypeMap("foo/bar", "json"))
	xCheck(t, ToJSON(rm.TypeMap) == "{\n  \"foo/bar\": \"json\"\n}",
		"bad:"+ToJSON(rm.TypeMap))

	xNoErr(t, rm.RemoveTypeMap("foo/bar"))
	xCheck(t, rm.TypeMap == nil, "should be nil")

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$meta",
		`{"contenttype":"bad/bad", "file": "foo"}`, 201, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "bad/bad",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$meta?inline=file", ``, 200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "bad/bad",
  "filebase64": "Zm9v",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xNoErr(t, rm.AddTypeMap("bad/bad", "json"))
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$meta?inline=file", ``, 200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "bad/bad",
  "file": "foo",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xNoErr(t, rm.RemoveTypeMap("bad/bad"))
	xNoErr(t, rm.AddTypeMap("bad/*", "json"))
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$meta?inline=file", ``, 200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "bad/bad",
  "file": "foo",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xNoErr(t, rm.AddTypeMap("bad/b*", "json"))
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$meta?inline=file", ``, 200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "bad/bad",
  "file": "foo",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xNoErr(t, rm.AddTypeMap("*/b*", "string"))
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$meta?inline=file", ``, 200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "bad/bad",
  "filebase64": "Zm9v",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xNoErr(t, rm.RemoveTypeMap("bad/*"))
	xNoErr(t, rm.RemoveTypeMap("bad/b*"))
	xNoErr(t, rm.RemoveTypeMap("bad/bad"))
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$meta?inline=file", ``, 200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "contenttype": "bad/bad",
  "file": "foo",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$meta",
		`{"file": "{\"foo\":\"bar\"}"}`,
		200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "bad/bad",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$meta?inline=file", ``, 200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "bad/bad",
  "file": "{\"foo\":\"bar\"}",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xNoErr(t, rm.AddTypeMap("*/b*", "json"))
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$meta?inline=file", ``, 200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "bad/bad",
  "file": {
    "foo": "bar"
  },

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xNoErr(t, rm.AddTypeMap("*/b*", "binary"))
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$meta?inline=file", ``, 200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "bad/bad",
  "filebase64": "eyJmb28iOiJiYXIifQ==",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$meta",
		`{"contenttype": null, "file": "foo\"bar"}`,
		200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$meta?inline=file", ``, 200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "filebase64": "Zm9vImJhcg==",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	// Force app/json to binary
	xNoErr(t, rm.AddTypeMap("application/json", "binary"))
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$meta",
		`{"file": "foo\"bar"}`,
		200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$meta?inline=file", ``, 200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",
  "filebase64": "Zm9vImJhcg==",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

	xNoErr(t, rm.RemoveTypeMap("application/json"))
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$meta?inline=file", ``, 200, `{
  "id": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1$meta",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "contenttype": "application/json",
  "file": "foo\"bar",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$meta",

  "versionscount": 1,
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
}
`)

}
