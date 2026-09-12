# Example output

Shape of a finished set of notes (Polish, from the `0.52.0` release-please PR).
Two sections shown; the real output covers every user-visible change.

---

# MarinaKeeper v0.52.0

Porównanie wersji: v0.51.0 → v0.52.0

## Najważniejsze zmiany

**Przekazania księgowe** (#390)
Eksport do księgowości przeniesiony z doraźnego okna do osobnej zakładki „Przekazania"
w widoku faktur. Przed utworzeniem przekazania widać listę blokad (np. armatorzy bez
identyfikatora księgowego, brak reguł podatkowych dla kraju podatkowego mariny),
pogrupowany podgląd tego, co trafi do księgowości, oraz wybór stawki podatku dla rat,
które nie mają własnej. Doszła lista utworzonych paczek z zamrożonymi wartościami —
ponowne pobranie daje ten sam plik — wraz ze statusami uzgodnienia. Utworzenie
przekazania ponownie sprawdza idempotencję przed podglądem, więc podwójne kliknięcie
nie utworzy duplikatów.

**Kontrolki daty zgodne z językiem oraz walidacja okresu umowy** (#391)
Natywne przeglądarkowe pola daty zostały zastąpione wspólnym date pickerem, który
formatuje i parsuje datę zgodnie z językiem użytkownika, a „dziś" wyznacza w czasie
lokalnym mariny. Daty rozpoczęcia i zakończenia umowy, terminy rat oraz daty aneksów
są walidowane po stronie interfejsu i serwera — m.in. „Termin raty musi mieścić się
w okresie umowy" i „Data obowiązywania wykracza poza okres umowy".

## Poprawki

- **i18n** (#392): poprawki tłumaczeń norweskich, rosyjskich, ukraińskich, szwedzkich
  i polskich — m.in. zastąpienie żargonu bazodanowego („rekord") słownictwem domenowym
  („wiersz") w polskich tekstach importu danych.

---

## What this example demonstrates

- The section title is the feature as a user would name it, not the commit subject.
- The first sentence says what changed for the user; specifics follow.
- UI strings are quoted (`„Przekazania"`, the validation messages), and domain terms
  come from `pl_PL.json` (*armator*, *paczka*, *stawka podatku*, *aneks*, *rata*).
- `(#390)` cites the PR; the compare link opens the release.
- Code-quality baselines, CI ratchet fixes, and version bumps from the same release
  are absent.
