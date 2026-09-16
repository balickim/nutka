// Package regularcontractapi translates authenticated contract HTTP commands into domain operations.
// It owns no persistence or business policy and receives both through explicit interfaces.
package regularcontractapi

import (
	"context"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

// Assignment identifies the two participants of one commercial relationship.
type Assignment struct {
	ID, TeacherID, LearnerID string
	Active                   bool
}

// ContractContext supplies the current scheduling state required by a command.
type ContractContext struct {
	Contract                  regularcontract.RegularContract
	Policy                    businesspolicy.Policy
	Availability, Unavailable []scheduling.Interval
	Lessons                   []scheduling.Lesson
}

// ActivationContext supplies assignment and availability data for activation.
type ActivationContext struct {
	Assignment                Assignment
	ContractID                string
	TeacherTimezone           string
	Policy                    businesspolicy.Policy
	Availability, Unavailable []scheduling.Interval
	Lessons                   []scheduling.Lesson
	CommercialState           commercial.OverlapState
}

// Transaction persists one aggregate mutation atomically with its history.
type Transaction interface {
	NewContractID(context.Context) (string, error)
	ActivationContext(context.Context, string) (ActivationContext, error)
	ContractContext(context.Context, string) (ContractContext, error)
	ConvertContractLessons(context.Context, string, []commercial.LessonReference) error
	SaveContract(context.Context, regularcontract.RegularContract) error
}

// Repository provides authorized reads and atomic mutation boundaries.
type Repository interface {
	Assignment(context.Context, string) (Assignment, error)
	Contracts(context.Context, string) ([]regularcontract.RegularContract, error)
	Contract(context.Context, string) (regularcontract.RegularContract, error)
	RunInTransaction(context.Context, func(Transaction) error) error
}

// Clock provides the UTC instant used by commands and horizon reads.
type Clock func() time.Time
