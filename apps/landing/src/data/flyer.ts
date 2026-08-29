/**
 * Treść ulotki A5 do druku (trasa /ulotka).
 *
 * Copy jest krótsze niż na stronie: kartkę ogląda się kilka sekund, a nie kilka minut.
 * Ulotka celowo nie podaje imienia i nazwiska — pierwszy plan zajmuje to, czego szuka
 * odbiorca (nauka gry), a nie to, kto uczy. Nazwisko i tak jest na stronie spod kodu QR.
 *
 * Sekwencje \u00A0 (spacja nierozdzielająca) wiążą ostatnie słowa zdań z tym, co po nich
 * następuje, żeby na końcu wiersza nie zostawały pojedyncze wyrazy wiszące. Zmiana treści
 * w okolicy takiej spacji wymaga ponownego spojrzenia na złamanie wierszy w PDF-ie.
 *
 * Kontakt i adres strony biorą się z ./config.ts — jedno źródło prawdy wspólne
 * z landingiem, żeby zmiana numeru telefonu nie wymagała pamiętania o druku.
 */

import { config } from "./config";

export const flyer = {
  siteUrl: config.siteUrl,
  contact: { ...config.contact },

  eyebrow: "Lekcje stacjonarne, indywidualnie",
  title: "Nauka gry na gitarze i keyboardzie",
  titleAccent: "dla początkujących.",

  lede: "Zaczynamy od zera i idziemy w Twoim tempie. Grasz dla własnej przyjemności, a\u00A0nie\u00A0dla\u00A0ocen.",

  points: [
    {
      claim: "Bez egzaminów i ocen",
      detail: "Postęp widać po tym, co potrafisz zagrać. Repertuar dobieramy razem — pod utwory, które\u00A0chcesz\u00A0umieć.",
    },
    {
      claim: "Talent to nie wszystko",
      detail: "Bez talentu też się nauczysz — najwięcej zależy od tego, ile ćwiczysz między lekcjami. Talent\u00A0przyspiesza początki nauki, nie całą resztę.",
    },
    {
      claim: "Na naukę nigdy nie jest za późno",
      detail: "Uczę dorosłych i seniorów. Pierwszy kontakt z instrumentem na emeryturze to normalny start, a\u00A0nie wyjątek.",
    },
    {
      claim: "Kwadrans dziennie wystarczy",
      detail: "Plan ćwiczeń układamy pod Twój tydzień. Po lekcji materiały — nagrania i nuty — czekają na\u00A0Twoim profilu na stronie.",
    },
  ],

  modes: ["Gitara", "Keyboard", "Indywidualnie", "Materiały po lekcji"],

  offer: {
    label: "Pierwsza lekcja próbna za darmo",
    detail: "Przychodzisz, sprawdzasz, czy to dla Ciebie. Bez opłaty i bez zobowiązania na dalsze zajęcia.",
  },

  contactLead: "Napisz lub zadzwoń.",
  qrCaption: "Zeskanuj — cennik i szczegóły",
} as const;

export type Flyer = typeof flyer;
