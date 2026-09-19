// Verifies that pieces stay inside one assignment, that a learner only adds and removes own wishes, and that materials link pieces of the same assignment.
package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
)

type pieceBody struct {
	ID               string  `json:"id"`
	Status           string  `json:"status"`
	StatusChangedAt  string  `json:"status_changed_at"`
	ProposedBy       string  `json:"proposed_by"`
	MaterialCount    int     `json:"material_count"`
	LatestMaterialAt *string `json:"latest_material_at"`
}

func TestPiecesFollowAuthorAndAssignmentRules(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedNamedTeacher(t, app, "second-teacher@example.test")
	seedTestLearner(t, app, true)
	seedNamedLearner(t, app, "other@example.test")
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, findID(t, app, authconfig.LearnersCollectionName, "learner@example.test"))
	otherAssignment := seedLessonAssignment(t, app, teacherID, findID(t, app, authconfig.LearnersCollectionName, "other@example.test"))
	teacher := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	stranger := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "second-teacher@example.test")
	learner := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	teacherPieces := "/api/teachers/assignments/" + assignment.Id + "/pieces"
	learnerPieces := "/api/learners/assignments/" + assignment.Id + "/pieces"
	create := func(path, body string, cookie *http.Cookie) pieceBody {
		t.Helper()
		response := request(t, server, http.MethodPost, path, body, cookie, true)
		var piece pieceBody
		if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &piece) != nil {
			t.Fatalf("create %s: %d %s", path, response.Code, response.Body.String())
		}
		return piece
	}
	status := func(name, method, path, body string, cookie *http.Cookie, want int) {
		t.Helper()
		if response := request(t, server, method, path, body, cookie, true); response.Code != want {
			t.Fatalf("%s: %d %s, want %d", name, response.Code, response.Body.String(), want)
		}
	}

	wish := create(learnerPieces, `{"title":" Hallelujah ","artist":"Cohen","status":"playing"}`, learner)
	if wish.Status != "wish" || wish.ProposedBy != "learner" || wish.StatusChangedAt != "2030-01-01T09:00:00Z" {
		t.Fatalf("learner wish: %+v", wish)
	}
	taught := create(teacherPieces, `{"title":"Oda do radości"}`, teacher)
	if taught.Status != "learning" || taught.ProposedBy != "teacher" {
		t.Fatalf("teacher piece: %+v", taught)
	}
	status("invalid status", http.MethodPost, teacherPieces, `{"title":"X","status":"done"}`, teacher, http.StatusBadRequest)
	status("empty title", http.MethodPost, teacherPieces, `{"title":"  "}`, teacher, http.StatusBadRequest)
	status("learner on teacher route", http.MethodPost, teacherPieces, `{"title":"X"}`, learner, http.StatusForbidden)
	status("foreign teacher update", http.MethodPatch, "/api/teachers/pieces/"+taught.ID, `{"status":"playing"}`, stranger, http.StatusForbidden)
	status("learner deletes teacher piece", http.MethodDelete, "/api/learners/pieces/"+taught.ID, "", learner, http.StatusConflict)

	now = now.Add(time.Hour)
	status("teacher starts wish", http.MethodPatch, "/api/teachers/pieces/"+wish.ID, `{"status":"learning"}`, teacher, http.StatusOK)
	status("learner deletes accepted wish", http.MethodDelete, "/api/learners/pieces/"+wish.ID, "", learner, http.StatusConflict)
	status("status goes back", http.MethodPatch, "/api/teachers/pieces/"+taught.ID, `{"status":"repertoire"}`, teacher, http.StatusOK)
	status("status goes back again", http.MethodPatch, "/api/teachers/pieces/"+taught.ID, `{"status":"learning"}`, teacher, http.StatusOK)

	own := create(learnerPieces, `{"title":"Perfect"}`, learner)
	status("learner deletes own wish", http.MethodDelete, "/api/learners/pieces/"+own.ID, "", learner, http.StatusNoContent)

	foreignPiece := create("/api/teachers/assignments/"+otherAssignment.Id+"/pieces", `{"title":"Obcy"}`, teacher)
	materialPath := "/api/teachers/assignments/" + assignment.Id + "/materials"
	if response := materialRequest(t, server, materialPath, map[string]string{"title": "Obcy", "body": "<p>x</p>", "piece": foreignPiece.ID}, nil, teacher); response.Code != http.StatusBadRequest {
		t.Fatalf("foreign piece on material: %d %s", response.Code, response.Body.String())
	}
	first := materialRequest(t, server, materialPath, map[string]string{"title": "Wersja 1", "body": "<p>x</p>", "piece": wish.ID}, nil, teacher)
	if first.Code != http.StatusCreated || !strings.Contains(first.Body.String(), `"piece":"`+wish.ID+`"`) {
		t.Fatalf("material with piece: %d %s", first.Code, first.Body.String())
	}
	loose := material(t, server, assignment.Id, teacher)
	status("pin foreign piece", http.MethodPatch, "/api/teachers/materials/"+loose, `{"piece":"`+foreignPiece.ID+`"}`, teacher, http.StatusBadRequest)
	status("pin material", http.MethodPatch, "/api/teachers/materials/"+loose, `{"piece":"`+wish.ID+`"}`, teacher, http.StatusOK)

	pieces := map[string]pieceBody{}
	for _, piece := range listPieces(t, server, learnerPieces, learner) {
		pieces[piece.ID] = piece
	}
	if len(pieces) != 2 || pieces[wish.ID].MaterialCount != 2 || pieces[wish.ID].LatestMaterialAt == nil || pieces[taught.ID].LatestMaterialAt != nil {
		t.Fatalf("learner list: %+v", pieces)
	}

	status("teacher deletes piece", http.MethodDelete, "/api/teachers/pieces/"+wish.ID, "", teacher, http.StatusNoContent)
	materials := request(t, server, http.MethodGet, "/api/learners/assignments/"+assignment.Id+"/materials", "", learner, false)
	if strings.Count(materials.Body.String(), `"piece":null`) != 2 {
		t.Fatalf("piece delete kept links or materials: %s", materials.Body.String())
	}

	assignment.Set(schedulingstore.ActiveField, false)
	if err := app.Save(assignment); err != nil {
		t.Fatal(err)
	}
	status("inactive assignment", http.MethodPost, learnerPieces, `{"title":"X"}`, learner, http.StatusConflict)
}

func listPieces(t *testing.T, server http.Handler, path string, cookie *http.Cookie) []pieceBody {
	t.Helper()
	response := request(t, server, http.MethodGet, path, "", cookie, false)
	var body struct {
		Items []pieceBody `json:"items"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &body) != nil {
		t.Fatalf("list: %d %s", response.Code, response.Body.String())
	}
	return body.Items
}
