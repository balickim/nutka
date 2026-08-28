/**
 * Treść ulotki A5 do druku (trasa /ulotka).
 *
 * Copy jest krótsze niż na stronie: kartkę ogląda się kilka sekund, a nie kilka minut.
 * Ulotka celowo nie podaje imienia i nazwiska — pierwszy plan zajmuje to, czego szuka
 * odbiorca (nauka gry), a nie to, kto uczy. Nazwisko i tak jest na stronie spod kodu QR.
 *
 * Kontakt i adres strony biorą się z ./config.ts — jedno źródło prawdy wspólne
 * z landingiem, żeby zmiana numeru telefonu nie wymagała pamiętania o druku.
 */

import { config } from "./config";

export const flyer = {
  siteUrl: config.siteUrl,
  contact: { ...config.contact },

  eyebrow: "Stacjonarnie i online",
  title: "Nauka gry na keyboardzie i gitarze",
  titleAccent: "dla początkujących.",

  lede: "Zaczynamy od zera i idziemy w Twoim tempie. Grasz dla własnej przyjemności, a nie dla ocen.",

  points: [
    {
      claim: "Bez egzaminów i ocen",
      detail: "Postęp widać po tym, co potrafisz zagrać. Repertuar dobieramy razem — pod utwory, które chcesz umieć.",
    },
    {
      claim: "Nie musisz mieć talentu",
      detail: "Gra to zestaw umiejętności rozłożonych na małe kroki. Talent przyspiesza pierwszy miesiąc, nie całą resztę.",
    },
    {
      claim: "Kwadrans dziennie wystarczy",
      detail: "Plan ćwiczeń układamy pod Twój tydzień, a nie pod wyobrażenie o tym, jak „porządnie” się ćwiczy.",
    },
  ],

  modes: ["Keyboard", "Gitara", "Stacjonarnie", "Online"],

  offer: {
    label: "Pierwsza lekcja próbna za darmo",
    detail: "Przychodzisz, sprawdzasz, czy to dla Ciebie. Bez opłaty i bez zobowiązania na dalsze zajęcia.",
  },

  contactLead: "Napisz lub zadzwoń.",
  qrCaption: "Zeskanuj — cennik i szczegóły",
} as const;

export type Flyer = typeof flyer;
