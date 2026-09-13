// This file contains request decoding and ownership helpers shared by scheduling handlers.
package schedulingapi

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"

	"github.com/pocketbase/pocketbase/core"
)

// lessonMutationMu serializes lesson writes within one process while the database transaction rechecks all conflicts.
var lessonMutationMu sync.Mutex

func bindBody(e *core.RequestEvent, destination any) error {
	if e.Request.Body == nil || e.Request.ContentLength == 0 {
		return errInvalid
	}
	raw, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return errInvalid
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return errInvalid
	}
	if err := ensureNoIdentityFields(fields); err != nil {
		return err
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return errInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return errInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errInvalid
	}
	return nil
}

func ownedRecord(app core.App, collection, id, field, account string) (*core.Record, error) {
	if id == "" {
		return nil, errInvalid
	}
	row, err := app.FindRecordById(collection, id)
	if err != nil {
		return nil, errForbidden
	}
	if row.GetString(field) != account {
		return nil, errForbidden
	}
	return row, nil
}
func deleteOwned(e *core.RequestEvent, collection string) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	err = e.App.RunInTransaction(func(tx core.App) error {
		row, lookupErr := ownedRecord(tx, collection, e.Request.PathValue("id"), "teacher", teacher.Id)
		if lookupErr != nil {
			return lookupErr
		}
		if err := tx.Delete(row); err != nil {
			return errInvalid
		}
		return nil
	})
	if err != nil {
		return handleError(e, err)
	}
	return e.JSON(204, nil)
}
