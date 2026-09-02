import type { Photo, PriceItem, Testimonial } from "./site";

/**
 * Dane zależne od konkretnej osoby i wdrożenia.
 * Uzupełnij wartości oznaczone prefiksem "TODO:" przed publikacją strony.
 */
export const config = {
  // Tymczasowy adres GitHub Pages. Zmień po wybraniu docelowej domeny.
  siteUrl: "https://balickim.github.io/nutka",
  appUrlFallback: "http://127.0.0.1:5173",
  teacherName: "Dominika Łuczyszyn",

  contact: {
    phone: "TODO: numer telefonu",
    phoneHref: "tel:+48000000000",
    email: "TODO: adres e-mail",
    emailHref: "mailto:kontakt@example.com",
    whatsappHref: "https://wa.me/48000000000",
    city: "TODO: miasto",
    address: "TODO: adres pracowni",
  },

  photos: {
    hero: null as Photo,
    portrait: null as Photo,
  },

  app: {
    available: false,
  },

  testimonials: {
    enabled: false,
    items: [] as Testimonial[],
  },

  pricing: {
    from: "TODO: cena",
    items: [
      { name: "Lekcja próbna", detail: "45 minut, stacjonarnie lub online", price: "TODO: cena" },
      { name: "Lekcja 60 minut", detail: "Stacjonarnie", price: "TODO: cena" },
      { name: "Lekcja 45 minut", detail: "Stacjonarnie", price: "TODO: cena" },
      { name: "Lekcja 60 minut", detail: "Online", price: "TODO: cena" },
      { name: "Pakiet 4 lekcji", detail: "Rozliczany z góry, ważny przez miesiąc", price: "TODO: cena" },
    ] as PriceItem[],
    rules: [
      "TODO: zasady odwoływania lekcji — do ilu godzin przed zajęciami i ile odwołań bez opłaty.",
      "TODO: forma płatności — gotówka, przelew, termin rozliczenia.",
      "TODO: czy jest dojazd do ucznia i w jakich okolicach.",
    ],
  },

} as const;
