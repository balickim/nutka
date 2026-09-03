import type { Photo, PriceItem, Testimonial } from "./site";

/**
 * Dane zależne od konkretnej osoby i wdrożenia.
 * Uzupełnij wartości oznaczone prefiksem "TODO:" przed publikacją strony.
 */
export const config = {
  // Tymczasowy adres GitHub Pages. Zmień po wybraniu docelowej domeny.
  siteUrl: "https://balickim.github.io/nutka",
  appUrlFallback: "http://127.0.0.1:5173",
  teacherName: "Dominika",

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
    from: "50 zł",
    items: [
      { name: "Bezpłatne spotkanie próbne", detail: "45 minut, bez zobowiązania", price: "0 zł" },
      { name: "Plan Regularny", detail: "Stały dzień i godzina, raz w tygodniu", price: "50 zł / 45 min" },
      { name: "Pakiet 4 zajęć", detail: "Ważny 60 dni, 65 zł za zajęcia", price: "260 zł" },
      { name: "Pojedyncza lekcja", detail: "Termin ustalany indywidualnie", price: "80 zł / 45 min" },
    ] as PriceItem[],
    rules: [
      "Plan Regularny opłacasz miesięcznie z góry, do 5. dnia miesiąca — kwota zależy od liczby zajęć w danym miesiącu.",
      "Pakiet 4 zajęć opłacasz z góry w całości. Jest ważny 60 dni od zakupu, a niewykorzystane zajęcia po tym czasie przepadają.",
      "Zajęcia przekładasz i odwołujesz bezpłatnie najpóźniej 24 godziny wcześniej. W Planie Regularnym masz jedno bezpłatne przełożenie w każdym miesiącu i dwa bezpłatne odwołania w okresie umowy.",
    ],
  },
} as const;
