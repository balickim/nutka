// Package paymentdetails normalizes and validates the bank transfer details of one teacher.
// It holds no storage or transport code. Callers read and write the teacher record fields it names.
package paymentdetails

import (
	"errors"
	"math/big"
	"strings"
	"unicode/utf8"
)

const (
	HolderField = "payment_account_holder"
	IBANField   = "payment_iban"
	NoteField   = "payment_note"

	HolderMaxLength = 140
	NoteMaxLength   = 300
	polishIBANSize  = 28
)

var ErrInvalid = errors.New("payment details are invalid")

// Details are the transfer values a teacher shows to the teacher's learners.
type Details struct {
	AccountHolder string `json:"account_holder"`
	IBAN          string `json:"iban"`
	Note          string `json:"note"`
}

// Record is the read surface of a stored teacher record.
type Record interface {
	GetString(string) string
}

// Normalize trims text, removes IBAN spaces, and adds the PL prefix to a 26-digit account number.
func Normalize(details Details) Details {
	iban := strings.ToUpper(strings.Join(strings.Fields(details.IBAN), ""))
	if len(iban) == polishIBANSize-2 && digitsOnly(iban) {
		iban = "PL" + iban
	}
	return Details{AccountHolder: strings.TrimSpace(details.AccountHolder), IBAN: iban, Note: strings.TrimSpace(details.Note)}
}

// Validate accepts empty details or a Polish IBAN with a holder. Call it on normalized details.
func Validate(details Details) error {
	if utf8.RuneCountInString(details.AccountHolder) > HolderMaxLength || utf8.RuneCountInString(details.Note) > NoteMaxLength {
		return ErrInvalid
	}
	if details.IBAN == "" {
		return nil
	}
	if details.AccountHolder == "" || !validPolishIBAN(details.IBAN) {
		return ErrInvalid
	}
	return nil
}

// FromRecord returns the stored details, or nil when the teacher has no IBAN.
func FromRecord(record Record) *Details {
	details := Details{AccountHolder: record.GetString(HolderField), IBAN: record.GetString(IBANField), Note: record.GetString(NoteField)}
	if details.IBAN == "" {
		return nil
	}
	return &details
}

// validPolishIBAN applies the ISO 13616 mod-97 checksum.
func validPolishIBAN(iban string) bool {
	if len(iban) != polishIBANSize || !strings.HasPrefix(iban, "PL") || !digitsOnly(iban[2:]) {
		return false
	}
	// Moving the country code and check digits to the end and mapping P=25, L=21 yields the checksum number.
	number, ok := new(big.Int).SetString(iban[4:]+"2521"+iban[2:4], 10)
	return ok && new(big.Int).Mod(number, big.NewInt(97)).Int64() == 1
}

func digitsOnly(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return value != ""
}
