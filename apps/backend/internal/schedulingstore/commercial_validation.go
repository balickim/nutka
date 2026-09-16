// Package schedulingstore validates commercial relations and ownership at the PocketBase record boundary.
// Validation prevents records from crossing teacher–learner assignments or violating token ownership.
package schedulingstore

import (
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/pocketbase/pocketbase/core"
)

func validateCommercialRecord(app core.App, record *core.Record) error {
	switch record.Collection().Name {
	case LessonPackagesCollectionName, RegularContractsCollectionName:
		return validatePolicySnapshot(record)
	case PackageTokensCollectionName:
		return validatePackageToken(app, record)
	case ContractAmendmentsCollectionName:
		if err := validateContractChild(app, record, ContractField); err != nil {
			return err
		}
		return validatePolicySnapshot(record)
	case ContractMonthsCollectionName:
		return validateContractMonth(app, record)
	case ChargesCollectionName:
		return validateChargeSource(app, record)
	case FinancialEntriesCollectionName:
		return validateFinancialEntry(app, record)
	case LessonsCollectionName:
		return validateLesson(app, record)
	case BusinessEventsCollectionName:
		return validateBusinessEvent(app, record)
	default:
		return nil
	}
}

func validateChargeSource(app core.App, charge *core.Record) error {
	sourceID := charge.GetString(SourceIDField)
	if sourceID == "" {
		return fmt.Errorf("charge %s must reference a source", charge.Id)
	}
	collections := map[string]string{"ad_hoc": LessonsCollectionName, "package": LessonPackagesCollectionName, "regular_contract": RegularContractsCollectionName}
	collection := collections[charge.GetString(SourceTypeField)]
	if collection == "" {
		return fmt.Errorf("charge %s has invalid source type %q", charge.Id, charge.GetString(SourceTypeField))
	}
	source, err := findRelated(app, collection, sourceID)
	if err != nil {
		return err
	}
	if assignment := source.GetString(AssignmentField); assignment != "" && assignment != charge.GetString(AssignmentField) {
		return fmt.Errorf("charge %s and source %s must share assignment", charge.Id, sourceID)
	}
	return nil
}

func validatePackageToken(app core.App, token *core.Record) error {
	packageRecord, err := findRelated(app, LessonPackagesCollectionName, token.GetString(PackageField))
	if err != nil {
		return err
	}
	lessonID := token.GetString(LessonField)
	state := token.GetString(TokenStateField)
	requiresLesson := state == "reserved" || state == "used"
	if requiresLesson != (lessonID != "") {
		return fmt.Errorf("package token %s state %q must match lesson relation", token.Id, state)
	}
	if lessonID == "" {
		return nil
	}
	lesson, err := findRelated(app, LessonsCollectionName, lessonID)
	if err != nil {
		return err
	}
	if lesson.GetString(AssignmentField) != packageRecord.GetString(AssignmentField) {
		return fmt.Errorf("package token %s and lesson %s must share assignment", token.Id, lesson.Id)
	}
	return nil
}

func validateContractChild(app core.App, record *core.Record, relation string) error {
	contract, err := findRelated(app, RegularContractsCollectionName, record.GetString(relation))
	if err != nil {
		return err
	}
	if assignment := record.GetString(AssignmentField); assignment != "" && assignment != contract.GetString(AssignmentField) {
		return fmt.Errorf("%s %s and contract %s must share assignment", record.Collection().Name, record.Id, contract.Id)
	}
	return nil
}

func validateContractMonth(app core.App, month *core.Record) error {
	contract, err := findRelated(app, RegularContractsCollectionName, month.GetString(ContractField))
	if err != nil {
		return err
	}
	chargeID := month.GetString(ChargeField)
	if chargeID == "" {
		return nil
	}
	charge, err := findRelated(app, ChargesCollectionName, chargeID)
	if err != nil {
		return err
	}
	if charge.GetString(AssignmentField) != contract.GetString(AssignmentField) {
		return fmt.Errorf("contract month %s and charge %s must share assignment", month.Id, charge.Id)
	}
	if charge.GetString(SourceTypeField) != "regular_contract" || charge.GetString(SourceIDField) != contract.Id || charge.GetString(PeriodField) != month.GetString(MonthField) {
		return fmt.Errorf("contract month %s charge source is inconsistent", month.Id)
	}
	return nil
}

func validateLesson(app core.App, lesson *core.Record) error {
	if lesson.Collection().Fields.GetByName(PlanTypeField) == nil {
		return nil
	}
	if lesson.GetInt(DurationMinutesField) != businesspolicy.LessonDurationMinutes {
		return fmt.Errorf("lessons must use a %d-minute duration", businesspolicy.LessonDurationMinutes)
	}
	if err := validatePolicySnapshot(lesson); err != nil {
		return err
	}
	assignment, err := findRelated(app, TeacherLearnersCollectionName, lesson.GetString(AssignmentField))
	if err != nil {
		return err
	}
	if lesson.GetString("teacher") != assignment.GetString("teacher") || lesson.GetString("learner") != assignment.GetString("learner") {
		return fmt.Errorf("lesson %s participants must match assignment %s", lesson.Id, assignment.Id)
	}
	if err := validateLessonToken(app, lesson); err != nil {
		return err
	}
	return validateLessonContract(app, lesson)
}

func validateLessonToken(app core.App, lesson *core.Record) error {
	tokenID := lesson.GetString(PackageTokenField)
	if tokenID == "" {
		return nil
	}
	token, err := findRelated(app, PackageTokensCollectionName, tokenID)
	if err != nil {
		return err
	}
	if token.GetString(PackageField) == "" || token.GetString(LessonField) != lesson.Id {
		return fmt.Errorf("lesson %s package token ownership is inconsistent", lesson.Id)
	}
	packageRecord, err := findRelated(app, LessonPackagesCollectionName, token.GetString(PackageField))
	if err != nil {
		return err
	}
	if packageRecord.GetString(AssignmentField) != lesson.GetString(AssignmentField) {
		return fmt.Errorf("lesson %s package token must share assignment", lesson.Id)
	}
	return nil
}

func validateLessonContract(app core.App, lesson *core.Record) error {
	contractID := lesson.GetString(ContractField)
	if contractID == "" {
		return nil
	}
	contract, err := findRelated(app, RegularContractsCollectionName, contractID)
	if err != nil {
		return err
	}
	if contract.GetString(AssignmentField) != lesson.GetString(AssignmentField) {
		return fmt.Errorf("lesson %s contract must share assignment", lesson.Id)
	}
	return nil
}

func validateBusinessEvent(app core.App, event *core.Record) error {
	aggregateType, aggregateID := event.GetString(AggregateTypeField), event.GetString(AggregateIDField)
	if aggregateID == "" {
		return nil
	}
	collectionByType := map[string]string{"lesson": LessonsCollectionName, "package": LessonPackagesCollectionName, "package_token": PackageTokensCollectionName, "contract": RegularContractsCollectionName, "contract_amendment": ContractAmendmentsCollectionName, "contract_month": ContractMonthsCollectionName, "charge": ChargesCollectionName, "financial_entry": FinancialEntriesCollectionName}
	collectionName := collectionByType[aggregateType]
	if collectionName == "" {
		return nil
	}
	aggregate, err := findRelated(app, collectionName, aggregateID)
	if err != nil {
		return err
	}
	assignment := aggregate.GetString(AssignmentField)
	if assignment == "" {
		assignment = aggregateAssignment(app, aggregateType, aggregate)
	}
	if assignment != "" && assignment != event.GetString(AssignmentField) {
		return fmt.Errorf("business event %s and aggregate %s must share assignment", event.Id, aggregateID)
	}
	return nil
}

func aggregateAssignment(app core.App, aggregateType string, aggregate *core.Record) string {
	switch aggregateType {
	case "package_token":
		packageRecord, err := findRelated(app, LessonPackagesCollectionName, aggregate.GetString(PackageField))
		if err == nil {
			return packageRecord.GetString(AssignmentField)
		}
	case "contract_amendment", "contract_month":
		contract, err := findRelated(app, RegularContractsCollectionName, aggregate.GetString(ContractField))
		if err == nil {
			return contract.GetString(AssignmentField)
		}
	case "financial_entry":
		charge, err := findRelated(app, ChargesCollectionName, aggregate.GetString(ChargeField))
		if err == nil {
			return charge.GetString(AssignmentField)
		}
	}
	return ""
}

func findRelated(app core.App, collection, id string) (*core.Record, error) {
	if id == "" {
		return nil, fmt.Errorf("missing required relation to %s", collection)
	}
	record, err := app.FindRecordById(collection, id)
	if err != nil {
		return nil, fmt.Errorf("find %s %s: %w", collection, id, err)
	}
	return record, nil
}
