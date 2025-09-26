package lynksearch_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lynkdb/lynkapi/go/codec"
	"github.com/lynkdb/lynkapi/go/lynkapi"
	"github.com/lynkdb/lynksearch/pkg/lynksearch"
)

var testDocuments = []map[string]any{
	{"content": "The quick brown fox jumps over the lazy dog"},
	{"content": "A quick brown dog jumps high"},
	{"content": "The lazy cat sleeps all day, run, running"},
	{"content": "中文测试 123"},
	{"content": "英文测试 123"},
	{"content": "测试测试"},
}

func Test_Main(t *testing.T) {

	tempdir := filepath.Clean(fmt.Sprintf("%s/lynksearch_test/", os.TempDir()))
	t.Logf("tempdir %s", tempdir)
	os.Mkdir(tempdir, 0775)

	spec := &lynkapi.TableSpec{}

	{
		spec.SetField("content", lynkapi.FieldSpec_String)
		spec.SetIndex("content", lynkapi.TableSpec_Index_FullTextSearch)

		js, _ := codec.Json.Encode(spec)
		t.Logf("TableSpec %s", string(js))
	}

	ins, err := lynksearch.NewInstance(lynksearch.InstanceConfig{
		Dir: tempdir,
	}, spec)
	if err != nil {
		t.Fatal(err)
	}

	for i, doc := range testDocuments {
		ins.AddDocument(fmt.Sprintf("%d", i+1), doc)
	}

	if err := ins.Flush(); err != nil {
		t.Fatal(err)
	}

	if rs := ins.Search("中文"); !rs.OK() {
		t.Fatal(rs.Error())
	} else if len(rs.Rows) == 0 {
		t.Fatal("data not found")
	} else if rs.Rows[0].Id != "4" {
		t.Fatal("search fail")
	} else {
		js, _ := json.Marshal(rs)
		t.Logf("result %s", string(js))
	}

	if rs := ins.Search("测试"); !rs.OK() {
		t.Fatal(rs.Error())
	} else if len(rs.Rows) != 3 {
		t.Fatal("data not match")
	} else if rs.Rows[0].Id != "6" {
		t.Fatal("search fail")
	} else {
		js, _ := json.Marshal(rs)
		t.Logf("result %s", string(js))
	}

}
