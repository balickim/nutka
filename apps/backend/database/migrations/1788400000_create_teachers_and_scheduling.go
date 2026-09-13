// Package migrations creates the teacher auth realm and scheduling storage.
// It preserves an existing custom users auth collection and is safe to rerun.
package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	teachersCollection          = authconfig.TeachersCollectionName
	legacyUsersCollection       = authconfig.LegacyUsersCollectionName
	learnersCollection          = authconfig.LearnersCollectionName
	teacherTimezoneField        = schedulingstore.TeacherTimezoneField
	defaultTeacherTimezone      = scheduling.DefaultTimezone
	teacherLearnersCollection   = schedulingstore.TeacherLearnersCollectionName
	availabilityRulesCollection = schedulingstore.AvailabilityRulesCollectionName
	availabilityExceptions      = schedulingstore.AvailabilityExceptionsCollectionName
	lessonsCollection           = schedulingstore.LessonsCollectionName
	lessonEventsCollection      = schedulingstore.LessonEventsCollectionName
)

func init() {
	migrations.Register(migrateTeachersAndScheduling, func(app core.App) error {
		return nil
	})
}

func migrateTeachersAndScheduling(app core.App) error {
	teachers, err := migrateTeachers(app)
	if err != nil {
		return err
	}
	learners, err := findCollection(app, learnersCollection)
	if err != nil {
		return fmt.Errorf("find %s collection: %w", learnersCollection, err)
	}
	if !learners.IsAuth() {
		return fmt.Errorf("%s collection must be an auth collection", learnersCollection)
	}
	if err := createSchedulingCollections(app, teachers, learners); err != nil {
		return err
	}
	return nil
}

func migrateTeachers(app core.App) (*core.Collection, error) {
	users, teachers, err := findTeacherRealms(app)
	if err != nil {
		return nil, err
	}
	if users != nil {
		if !users.IsAuth() {
			return nil, fmt.Errorf("cannot migrate %s to %s: collection is not an auth collection", legacyUsersCollection, teachersCollection)
		}
		users.Name = teachersCollection
		teachers = users
	}
	if teachers == nil {
		teachers = core.NewAuthCollection(teachersCollection)
	}
	if !teachers.IsAuth() {
		return nil, fmt.Errorf("%s collection must be an auth collection", teachersCollection)
	}
	configureTeacherAuth(teachers)
	if err := addTeacherFields(teachers); err != nil {
		return nil, err
	}
	if err := app.Save(teachers); err != nil {
		return nil, fmt.Errorf("save %s collection: %w", teachersCollection, err)
	}
	if err := setTeacherTimezoneDefaults(app, teachers); err != nil {
		return nil, err
	}
	return teachers, nil
}

func findTeacherRealms(app core.App) (*core.Collection, *core.Collection, error) {
	users, err := findOptionalCollection(app, legacyUsersCollection)
	if err != nil {
		return nil, nil, err
	}
	teachers, err := findOptionalCollection(app, teachersCollection)
	if err != nil {
		return nil, nil, err
	}
	if users != nil && teachers != nil {
		return nil, nil, fmt.Errorf("cannot migrate %s to %s: both collections exist with incompatible data", legacyUsersCollection, teachersCollection)
	}
	return users, teachers, nil
}

func findOptionalCollection(app core.App, name string) (*core.Collection, error) {
	collection, err := findCollection(app, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect %s collection: %w", name, err)
	}
	return collection, nil
}

func configureTeacherAuth(collection *core.Collection) {
	collection.ListRule = nil
	collection.ViewRule = nil
	collection.CreateRule = nil
	collection.UpdateRule = nil
	collection.DeleteRule = nil
	collection.AuthRule = types.Pointer("")
	collection.AuthToken.Duration = int64(authconfig.SessionDuration / time.Second)
	collection.PasswordAuth.Enabled = true
	collection.PasswordAuth.IdentityFields = []string{core.FieldNameEmail}
	collection.MFA.Enabled = false
	collection.OTP.Enabled = false
	collection.OAuth2.MappedFields.Name = "name"
	collection.OAuth2.MappedFields.AvatarURL = "avatar"
}

func addTeacherFields(collection *core.Collection) error {
	if field := collection.Fields.GetByName("name"); field == nil {
		collection.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 255, Presentable: true})
	} else if name, ok := field.(*core.TextField); !ok {
		return fmt.Errorf("%s.name must be a text field", teachersCollection)
	} else {
		name.Required = true
		name.Max = 255
		name.Presentable = true
	}
	if collection.Fields.GetByName("last_login_at") == nil {
		collection.Fields.Add(&core.DateField{Name: "last_login_at", Hidden: true})
	}
	if collection.Fields.GetByName("login_metadata") == nil {
		collection.Fields.Add(&core.JSONField{Name: "login_metadata", Hidden: true, MaxSize: 16 * 1024})
	}
	if field := collection.Fields.GetByName("avatar"); field == nil {
		collection.Fields.Add(&core.FileField{
			Name:      "avatar",
			MaxSelect: 1,
			MimeTypes: []string{"image/jpeg", "image/png", "image/svg+xml", "image/gif", "image/webp"},
		})
	} else if _, ok := field.(*core.FileField); !ok {
		return fmt.Errorf("%s.avatar must be a file field", teachersCollection)
	} else {
		avatar := field.(*core.FileField)
		avatar.MaxSelect = 1
		avatar.MimeTypes = []string{"image/jpeg", "image/png", "image/svg+xml", "image/gif", "image/webp"}
	}
	if field := collection.Fields.GetByName(teacherTimezoneField); field == nil {
		collection.Fields.Add(&core.TextField{
			Name:     teacherTimezoneField,
			Required: true,
			Max:      128,
			Pattern:  `^[A-Za-z0-9._+~-]+(?:/[A-Za-z0-9._+~-]+)*$`,
		})
	} else if timezone, ok := field.(*core.TextField); !ok {
		return fmt.Errorf("%s.%s must be a text field", teachersCollection, teacherTimezoneField)
	} else {
		timezone.Required = true
		timezone.Max = 128
		timezone.Pattern = `^[A-Za-z0-9._+~-]+(?:/[A-Za-z0-9._+~-]+)*$`
	}
	return nil
}

func setTeacherTimezoneDefaults(app core.App, collection *core.Collection) error {
	records, err := app.FindAllRecords(collection)
	if err != nil {
		return fmt.Errorf("read %s records: %w", teachersCollection, err)
	}
	for _, record := range records {
		timezone := record.GetString(teacherTimezoneField)
		if timezone == "" {
			record.Set(teacherTimezoneField, defaultTeacherTimezone)
			if err := app.Save(record); err != nil {
				return fmt.Errorf("set timezone for teacher %s: %w", record.Id, err)
			}
		} else if _, err := time.LoadLocation(timezone); err != nil {
			return fmt.Errorf("invalid timezone for teacher %s: %q", record.Id, timezone)
		}
	}
	return nil
}
