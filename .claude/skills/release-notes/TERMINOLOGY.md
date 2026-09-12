# Terminology

Release notes must name things the way the UI names them. A reader who sees
"właściciel" in the notes and "Armator" in the app has to translate for you.

## Method

1. Find the module's locale namespace: `translation.<module>.*` mirrors
   `sailormoon.ui/src/modules/<module>/`.
2. Dump the module's own keys, plus its `menu.*` entry for the canonical module name:
   ```bash
   $SKILL/scripts/locale-terms.py pl_PL '^translation\.invoices\.handoffs\.'
   $SKILL/scripts/locale-terms.py pl_PL '^translation\.menu\.'
   ```
3. For a term you already have in mind, check whether the UI actually uses it before
   committing to it:
   ```bash
   $SKILL/scripts/locale-terms.py pl_PL 'paczka|partia' --values
   ```
4. Prefer the tab, dialog, and button labels over your own paraphrase. When a
   validation message states the new rule, quote it.

## Polish (pl_PL) — terms that are easy to get wrong

| Concept | Use | Not |
| --- | --- | --- |
| boat owner | **armator**, *Armatorzy* | właściciel |
| owner portal | **Portal armatora** | portal właściciela |
| berth | **stanowisko** | miejsce, keja |
| accounting handoff | **przekazanie księgowe**, tab *Przekazania* | eksport księgowy (the removed dialog) |
| handoff batch | **paczka** | partia |
| reconciliation | **uzgadnianie**, *Uzgodnione* | rekoncyliacja, rozliczenie |
| tax rate | **stawka podatku** | stawka VAT |
| contract amendment | **aneks** (verb: *Aneksuj*) | poprawka, zmiana umowy |
| installment | **rata**, *Harmonogram rat* | transza |
| installment due date | **termin raty** | data płatności raty |
| contract term | **okres umowy** | termin umowy |
| marina setup card | **Konfiguracja mariny** | onboarding, lista kontrolna |
| customer wizard | **kreator klienta mariny**, CTA *Wdróż klienta* | asystent klienta |
| marina-local time | **czas lokalny mariny** | strefa czasowa mariny |
| amount in words | **kwota słownie** | kwota tekstem |
| product name | **MarinaKeeper** | Sailormoon (repository only) |

Module names: *Dashboard, Armatorzy, Łodzie, Stanowiska, Mapa Mariny, Umowy, Faktury,
Płatności, Opłaty dodatkowe, Podnajem, Powiadomienia, Szablony, Kolejka działań,
Aktywność użytkowników, Ustawienia, Portal armatora*.

Note: `pl_PL.json` uses **armator** everywhere for the boat owner. The single
exception is `translation.auth.unified.owner_card_badge` ("Dla armatorów i właścicieli
jachtów"), where the pair is deliberate landing-page wording. Treat any other
"właściciel" you find as a bug in the locale file, not as a term to copy.

## Other languages

No curated table yet. Build one the same way for the locale you are writing in
(`da_DK`, `de_DE`, `nb_NO`, `sv_SE`, `uk_UA`, `ru_RU`), and add the traps you hit to
this file so the next release does not repeat the work.
