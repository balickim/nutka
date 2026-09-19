// Verifies that learner materials stay scoped to one assignment and that uploads are sanitized and type-checked.
package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/pocketbase/pocketbase/core"
)

type materialFile struct {
	name    string
	content []byte
}

func materialRequest(t *testing.T, server http.Handler, path string, fields map[string]string, files []materialFile, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := form.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range files {
		part, err := form.CreateFormFile("attachments", file.name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write(file.content)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set(authconfig.AuthIntentHeader, authconfig.AuthIntentValue)
	req.AddCookie(cookie)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, req)
	return response
}

func seedVerified(t *testing.T, app core.App, collection, email string) string {
	t.Helper()
	realm, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(realm)
	record.SetEmail(email)
	record.Set("name", email)
	record.SetPassword("local-password")
	record.SetVerified(true)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record.Id
}

func TestLearnerMaterialsAreScopedToOneAssignment(t *testing.T) {
	app, server := newTestServer(t)
	teacherID := seedVerified(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := seedVerified(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	otherLearnerID := seedVerified(t, app, authconfig.LearnersCollectionName, "other@example.test")
	seedVerified(t, app, authconfig.TeachersCollectionName, "stranger@example.test")
	assignment := seedAssignment(t, app, teacherID, learnerID)
	seedAssignment(t, app, teacherID, otherLearnerID)
	teacher := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	stranger := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "stranger@example.test")
	learner := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	other := loginCookie(t, server, "/api/collections/learners/auth-with-password", "other@example.test")
	createPath := "/api/teachers/assignments/" + assignment.Id + "/materials"
	pdf := materialFile{name: "gamy.pdf", content: []byte("%PDF-1.4\n%%EOF\n")}
	fields := map[string]string{"title": "Gamy", "body": `<p>Ćwicz <strong>C-dur</strong><script>alert(1)</script></p>`}

	if response := materialRequest(t, server, createPath, fields, []materialFile{pdf}, stranger); response.Code != http.StatusForbidden {
		t.Fatalf("foreign teacher created a material: %d %s", response.Code, response.Body.String())
	}
	if response := materialRequest(t, server, createPath, fields, []materialFile{{name: "run.sh", content: []byte("#!/bin/sh\necho x\n")}}, teacher); response.Code != http.StatusBadRequest {
		t.Fatalf("disallowed file type was accepted: %d %s", response.Code, response.Body.String())
	}
	response := materialRequest(t, server, createPath, fields, []materialFile{pdf}, teacher)
	if response.Code != http.StatusCreated || strings.Contains(response.Body.String(), "script") {
		t.Fatalf("teacher create: %d %s", response.Code, response.Body.String())
	}
	var created struct {
		Attachments []struct{ URL string } `json:"attachments"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil || len(created.Attachments) != 1 {
		t.Fatalf("created material shape: %v %s", err, response.Body.String())
	}
	learnerFile := strings.Replace(created.Attachments[0].URL, "/api/teachers/", "/api/learners/", 1)

	listPath := "/api/learners/assignments/" + assignment.Id + "/materials"
	if response := request(t, server, http.MethodGet, listPath, "", learner, false); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "C-dur") {
		t.Fatalf("learner list: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodGet, listPath, "", other, false); response.Code != http.StatusForbidden {
		t.Fatalf("other learner listed materials: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodGet, learnerFile, "", learner, false); response.Code != http.StatusOK || !strings.HasPrefix(response.Body.String(), "%PDF") || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("learner file: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodGet, learnerFile, "", other, false); response.Code != http.StatusForbidden {
		t.Fatalf("other learner read a file: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodGet, created.Attachments[0].URL, "", learner, false); response.Code != http.StatusForbidden {
		t.Fatalf("learner session crossed teacher file realm: %d %s", response.Code, response.Body.String())
	}
	nativeFile := strings.Replace(strings.Replace(learnerFile, "/api/learners/materials/", "/api/files/learner_materials/", 1), "/files/gamy", "/gamy", 1)
	for _, native := range []string{"/api/collections/learner_materials/records", nativeFile} {
		if response := request(t, server, http.MethodGet, native, "", learner, false); response.Code == http.StatusOK && !strings.Contains(response.Body.String(), `"items":[]`) {
			t.Fatalf("native route exposed materials: %s %d %s", native, response.Code, response.Body.String())
		}
	}
}
