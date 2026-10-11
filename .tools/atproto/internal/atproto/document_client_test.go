package atproto

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestDocumentWireUsesExplicitNullAndCIDForConditionalMutations(t *testing.T) {
	client, base := newFake(t)
	uri := "at://" + DID + "/" + DocumentCollection + "/3fcjynoduk2zd"
	value := map[string]any{"$type": DocumentCollection, "site": "at://" + DID + "/" + PublicationCollection + "/" + siteKey, "path": "/posts/example/", "title": "Example", "publishedAt": "2018-03-28T23:26:00+02:00", "textContent": "Complete body", "tags": []string{"lang:ca"}}
	written := testRecord("3fcjynoduk2zd", value)
	written.URI = uri
	var inputs []map[string]any
	client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		method := strings.TrimPrefix(request.URL.Path, "/xrpc/")
		if method != "com.atproto.repo.putRecord" && method != "com.atproto.repo.deleteRecord" {
			return base.RoundTrip(request)
		}
		if request.Method != http.MethodPost || request.Header.Get("Authorization") != "Bearer "+access {
			t.Fatal("document mutation missing correct method/session")
		}
		var input map[string]any
		decoder := json.NewDecoder(request.Body)
		decoder.UseNumber()
		if err := decoder.Decode(&input); err != nil {
			t.Fatal(err)
		}
		if input["repo"] != DID || input["collection"] != DocumentCollection || input["rkey"] != "3fcjynoduk2zd" {
			t.Fatal("document mutation target differs")
		}
		if _, present := input["swapRecord"]; !present {
			t.Fatal("conditional create omitted null swapRecord")
		}
		if len(inputs) == 0 && input["swapRecord"] != nil {
			t.Fatal("create did not send JSON null")
		}
		if len(inputs) > 0 && input["swapRecord"] != written.CID {
			t.Fatal("update/delete omitted prior CID")
		}
		inputs = append(inputs, input)
		if method == "com.atproto.repo.deleteRecord" {
			return jsonResponse(200, map[string]any{}), nil
		}
		return jsonResponse(200, map[string]any{"uri": written.URI, "cid": written.CID}), nil
	})
	err := client.WithDocumentSession(context.Background(), Credentials{Handle: Handle, AppPassword: password}, func(writer DocumentWriter) error {
		created, err := writer.PutDocument(context.Background(), Record{URI: uri, Value: value})
		if err != nil {
			return err
		}
		created.Value = value
		if _, err := writer.PutDocument(context.Background(), created); err != nil {
			return err
		}
		return writer.DeleteDocument(context.Background(), created)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 3 || count(base.events, "com.atproto.server.createSession") != 1 || count(base.events, "com.atproto.server.deleteSession") != 1 {
		t.Fatal("document batch did not share one cleaned session")
	}
}

func TestDocumentMutationRejectsInvalidAcknowledgementAndNullDeleteOutput(t *testing.T) {
	for _, kind := range []string{"wrong URI", "null deletion"} {
		t.Run(kind, func(t *testing.T) {
			client, base := newFake(t)
			client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if strings.HasSuffix(request.URL.Path, "deleteRecord") {
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader("null"))}, nil
				}
				if strings.HasSuffix(request.URL.Path, "putRecord") {
					return jsonResponse(200, map[string]any{"uri": "at://" + DID + "/" + DocumentCollection + "/3mxmtmkzqgp2t", "cid": testRecord(siteKey, DesiredPublication()).CID}), nil
				}
				return base.RoundTrip(request)
			})
			err := client.WithDocumentSession(context.Background(), Credentials{Handle: Handle, AppPassword: password}, func(writer DocumentWriter) error {
				r := Record{URI: "at://" + DID + "/" + DocumentCollection + "/3fcjynoduk2zd", CID: testRecord(siteKey, DesiredPublication()).CID}
				if kind == "null deletion" {
					return writer.DeleteDocument(context.Background(), r)
				}
				_, err := writer.PutDocument(context.Background(), r)
				return err
			})
			if err == nil || count(base.events, "com.atproto.server.deleteSession") != 1 {
				t.Fatal("invalid acknowledgement accepted or session cleanup omitted")
			}
		})
	}
}

func TestPinnedDocumentSchemaAndXRPCContracts(t *testing.T) {
	data, err := os.ReadFile("testdata/site.standard.document.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		CID   string `json:"cid"`
		Value struct {
			ID   string `json:"id"`
			Defs map[string]struct {
				Key    string `json:"key"`
				Record struct {
					Required   []string `json:"required"`
					Properties map[string]struct {
						Type, Format            string
						MaxLength, MaxGraphemes int
					} `json:"properties"`
				} `json:"record"`
			} `json:"defs"`
		} `json:"value"`
	}
	if json.Unmarshal(data, &fixture) != nil || fixture.CID != "bafyreifylc5lfr4vqfzrwdtqm4pnfg7cbra7z4acbgikylqandzefwdeaq" || fixture.Value.ID != DocumentCollection {
		t.Fatal("wrong document schema fixture")
	}
	main := fixture.Value.Defs["main"]
	if main.Key != "tid" || !reflect.DeepEqual(main.Record.Required, []string{"site", "title", "publishedAt"}) || main.Record.Properties["textContent"].Type != "string" || main.Record.Properties["publishedAt"].Format != "datetime" || main.Record.Properties["title"].MaxLength != 5000 || main.Record.Properties["title"].MaxGraphemes != 500 {
		t.Fatal("document schema contract changed")
	}
	for _, method := range []string{"putRecord", "deleteRecord", "getRecord", "listRecords"} {
		data, err := os.ReadFile("testdata/com.atproto.repo." + method + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var api struct {
			ID   string
			Defs map[string]struct {
				Type  string
				Input struct {
					Schema struct {
						Required, Nullable []string
						Properties         map[string]struct{ Type, Format string }
					}
				}
				Parameters struct{ Required []string }
			} `json:"defs"`
		}
		if json.Unmarshal(data, &api) != nil || api.ID != "com.atproto.repo."+method {
			t.Fatal("wrong XRPC fixture")
		}
		main := api.Defs["main"]
		if method == "putRecord" && (main.Type != "procedure" || !reflect.DeepEqual(main.Input.Schema.Nullable, []string{"swapRecord"}) || main.Input.Schema.Properties["swapRecord"].Format != "cid") {
			t.Fatal("conditional put contract changed")
		}
		if method == "deleteRecord" && (main.Type != "procedure" || main.Input.Schema.Properties["swapRecord"].Format != "cid") {
			t.Fatal("conditional delete contract changed")
		}
		if (method == "getRecord" || method == "listRecords") && main.Type != "query" {
			t.Fatal("public read contract changed")
		}
	}
}
