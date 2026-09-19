package paymentdetails

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeAndValidate(t *testing.T) {
	cases := []struct {
		name  string
		input Details
		iban  string
		err   error
	}{
		{"spaced account without prefix", Details{AccountHolder: " Dominika Nowak ", IBAN: "61 1090 1014 0000 0712 1981 2874"}, "PL61109010140000071219812874", nil},
		{"lower-case prefix", Details{AccountHolder: "Dominika", IBAN: "pl61109010140000071219812874"}, "PL61109010140000071219812874", nil},
		{"empty details", Details{}, "", nil},
		{"wrong checksum", Details{AccountHolder: "Dominika", IBAN: "PL61109010140000071219812875"}, "PL61109010140000071219812875", ErrInvalid},
		{"foreign country", Details{AccountHolder: "Dominika", IBAN: "DE89370400440532013000"}, "DE89370400440532013000", ErrInvalid},
		{"iban without holder", Details{IBAN: "PL61109010140000071219812874"}, "PL61109010140000071219812874", ErrInvalid},
		{"note too long", Details{Note: strings.Repeat("ą", NoteMaxLength+1)}, "", ErrInvalid},
	}
	for _, tc := range cases {
		normalized := Normalize(tc.input)
		if normalized.IBAN != tc.iban {
			t.Fatalf("%s: iban %q, want %q", tc.name, normalized.IBAN, tc.iban)
		}
		if err := Validate(normalized); !errors.Is(err, tc.err) {
			t.Fatalf("%s: err %v, want %v", tc.name, err, tc.err)
		}
	}
}

type record map[string]string

func (r record) GetString(key string) string { return r[key] }

func TestFromRecordHidesDetailsWithoutIBAN(t *testing.T) {
	if FromRecord(record{HolderField: "Dominika"}) != nil {
		t.Fatal("details without an IBAN were exposed")
	}
	details := FromRecord(record{HolderField: "Dominika", IBANField: "PL61109010140000071219812874", NoteField: "Gotówka też"})
	if details == nil || details.Note != "Gotówka też" {
		t.Fatalf("stored details were not returned: %+v", details)
	}
}
