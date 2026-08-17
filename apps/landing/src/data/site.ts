/**
 * Jedyne źródło treści i danych dla landingu oraz podstrony /cennik.
 *
 * Zasada: nic zmyślonego. Każda wartość, której jeszcze nie znamy, ma prefiks "TODO:"
 * i w tej postaci jest widoczna na stronie — dzięki temu nieuzupełnione pole rzuca się
 * w oczy zamiast po cichu trafić na produkcję jako wymyślony fakt.
 *
 * Copy pisane jest w czasie teraźniejszym i formach bezosobowych. Polski czas przeszły
 * jest rodzajowy ("zaczynałeś" / "zaczynałaś"), a strona mówi do wszystkich odbiorców naraz.
 */

export type Photo = string | null;

export interface Testimonial {
  quote: string;
  author: string;
}

export interface PriceItem {
  name: string;
  detail: string;
  price: string;
}

const teacherName = "Dominika Łuczyszyn";

export const site = {
  teacherName,

  meta: {
    title: `${teacherName} — lekcje pianina, keyboardu i gitary dla dorosłych`,
    description:
      "Lekcje pianina, keyboardu i gitary dla dorosłych, którzy chcą grać dla własnej przyjemności. Bez egzaminów, ocen i presji, stacjonarnie i online. Umów lekcję próbną.",
    ogImage: "/og.png",
  },

  contact: {
    // TODO: uzupełnić prawdziwe dane kontaktowe.
    phone: "TODO: numer telefonu",
    phoneHref: "tel:+48000000000",
    email: "TODO: adres e-mail",
    emailHref: "mailto:kontakt@example.com",
    whatsappHref: "https://wa.me/48000000000",
    city: "TODO: miasto",
    address: "TODO: adres pracowni",
  },

  nav: [
    { label: "Lekcje", href: "/#lekcje" },
    { label: "O mnie", href: "/#o-mnie" },
    { label: "Cennik", href: "/cennik" },
    { label: "FAQ", href: "/#faq" },
  ],

  hero: {
    eyebrow: "Pianino, keyboard i gitara · stacjonarnie i online",
    title: "To nie jest szkoła muzyczna.",
    titleAccent: "Tu grasz dla siebie.",
    lede: "Bez egzaminów, ocen i presji. Uczę dorosłych, którzy chcą grać dla własnej przyjemności — w swoim tempie i na swoich warunkach.",
    ctaPrimary: "Umów lekcję próbną",
    ctaSecondary: "Zobacz, jak zacząć",
    photo: null as Photo,
  },

  objections: {
    eyebrow: "Trzy rzeczy, które zwykle zatrzymują",
    title: "Nic z tego nie jest przeszkodą.",
    items: [
      {
        claim: "„Jestem za stary, żeby zaczynać”",
        answer:
          "Dorosły uczeń ma przewagę, o której rzadko się mówi: wie, po co tu jest. Sam wybiera utwory, rozumie, co ćwiczy, i nie musi nikomu niczego udowadniać.",
      },
      {
        claim: "„Nie mam talentu”",
        answer:
          "Gra na instrumencie to zestaw umiejętności, które da się rozłożyć na małe kroki i przejść po kolei. Talent przyspiesza pierwszy miesiąc, a nie decyduje o całej reszcie.",
      },
      {
        claim: "„Nie mam czasu ćwiczyć”",
        answer:
          "Kwadrans dziennie daje więcej niż trzy godziny raz w tygodniu. Plan układamy pod Twój tydzień, a nie pod wyobrażenie o tym, jak „porządnie” powinno się ćwiczyć.",
      },
    ],
  },

  instruments: {
    eyebrow: "Lekcje",
    // Nagłówek celowo nie liczy instrumentów: karta klawiszowa obejmuje i pianino, i keyboard.
    title: "Klawisze albo gitara. Jeden powód: chcesz grać.",
    items: [
      {
        name: "Pianino i keyboard",
        symbol: "♪",
        description:
          "Od pierwszego akordu po utwór, który chodzi Ci po głowie od lat. Klasyka, pop, muzyka filmowa — repertuar wybieramy razem. Do ćwiczeń w domu keyboard w zupełności wystarczy.",
      },
      {
        name: "Gitara klasyczna i akustyczna",
        symbol: "♫",
        description:
          "Akordy do śpiewania w gronie znajomych, palcówki, ulubione utwory — zaczynamy od tego, co chcesz zagrać.",
      },
    ],
    modes: ["Stacjonarnie", "Online", "Dla początkujących", "Dla powracających"],
  },

  about: {
    eyebrow: "O mnie",
    title: "Uczę tak, żeby chciało się wrócić do instrumentu w kolejnym tygodniu.",
    paragraphs: [
      "Od wielu lat uczę gry na pianinie, keyboardzie i gitarze — dzieci i dorosłych. Praca z dziećmi nauczyła mnie rozkładania trudnych rzeczy na najmniejsze możliwe kroki. Praca z dorosłymi — tego, że o powodzeniu decyduje nie talent, tylko to, czy lekcja daje frajdę.",
      "TODO: wykształcenie muzyczne, ukończone szkoły, doświadczenie sceniczne.",
      "TODO: zdanie osobiste — dlaczego uczysz i co najbardziej cieszy Cię w tej pracy.",
    ],
    portrait: null as Photo,
  },

  howItWorks: {
    eyebrow: "Jak zacząć",
    title: "Cztery kroki do pierwszej lekcji.",
    steps: [
      {
        name: "Piszesz albo dzwonisz",
        description: "Jedna wiadomość wystarczy. Nie musisz wcześniej wiedzieć, czego chcesz się uczyć.",
      },
      {
        name: "Rozmawiamy o tym, co chcesz grać",
        description: "Ustalamy instrument, cele i realny czas, jaki masz w tygodniu.",
      },
      {
        name: "Lekcja próbna",
        description: "Sprawdzasz, czy to dla Ciebie. Bez zobowiązania na dalsze zajęcia.",
      },
      {
        name: "Ruszamy w Twoim tempie",
        description: "Dostajesz plan i materiały do ćwiczeń, a terminy dopasowujemy do Twojego tygodnia.",
      },
    ],
  },

  app: {
    // TODO: gdy aplikacja będzie dostępna dla uczniów, usunąć badge i zmienić czas na teraźniejszy.
    available: false,
    badge: "W przygotowaniu",
    eyebrow: "Aplikacja dla uczniów",
    title: "Wszystko z lekcji w jednym miejscu.",
    lede: "Powstaje aplikacja, która ma domykać to, co dzieje się między zajęciami.",
    features: [
      { name: "Nagrania", description: "Wracasz do materiału z lekcji, kiedy tylko chcesz." },
      { name: "Zadania", description: "Wiesz dokładnie, co ćwiczyć w danym tygodniu." },
      { name: "Terminy", description: "Umawiasz i przekładasz zajęcia bez dzwonienia." },
    ],
  },

  testimonials: {
    // Zostaje wyłączone dopóki nie pojawią się prawdziwe wypowiedzi uczniów.
    // Zmyślone opinie z imionami wprowadzają klienta w błąd — sekcja włącza się jedną linią.
    enabled: false,
    eyebrow: "Opinie",
    title: "Co mówią uczniowie.",
    items: [] as Testimonial[],
  },

  pricing: {
    eyebrow: "Cennik",
    title: "Jasne stawki, bez ukrytych kosztów.",
    from: "TODO: cena",
    fromUnit: "za 60 minut",
    teaserNote: "Lekcja próbna jest krótsza i tańsza — sprawdzasz bez zobowiązań.",
    ctaLabel: "Zobacz pełny cennik",
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

  faq: {
    eyebrow: "FAQ",
    title: "Pytania, które padają najczęściej.",
    items: [
      {
        question: "Czy uczysz też dzieci?",
        answer:
          "Tak. Od lat uczę zarówno dzieci, jak i dorosłych — ta strona opisuje zajęcia dla dorosłych, ale jeśli szukasz lekcji dla dziecka, po prostu napisz.",
      },
      {
        question: "Nie mam instrumentu w domu. Co wtedy?",
        answer:
          "Na pierwszą lekcję instrument nie jest potrzebny, ale do ćwiczenia w domu już tak. Przy klawiszach na start w zupełności wystarczy keyboard z ważoną klawiaturą — na pianino zawsze zdążysz. Przy gitarze początkującym polecam klasyczną: nylonowe struny są łagodniejsze dla nieprzyzwyczajonych palców, a szerszy gryf wybacza więcej przy ustawianiu ręki. Chętnie doradzę przed zakupem, żeby nie przepłacić.",
      },
      {
        question: "Jak wyglądają lekcje online?",
        answer:
          "Tak samo jak stacjonarne, tylko przez kamerę. Potrzebujesz telefonu albo laptopa i ustawienia kamery tak, żeby było widać ręce i instrument.",
      },
      {
        question: "Czy muszę znać nuty?",
        answer:
          "Nie. Nut uczysz się w takim zakresie, w jakim są Ci realnie potrzebne do grania. Można też pracować na akordach i tabulaturach.",
      },
      {
        question: "Jak często odbywają się lekcje?",
        answer:
          "Standardowo raz w tygodniu — w tym rytmie postępy są najbardziej widoczne. Jeśli Twój grafik na to nie pozwala, ustalimy inny.",
      },
      {
        question: "Co, jeśli muszę odwołać lekcję?",
        answer: "TODO: zasady odwoływania lekcji.",
      },
    ],
  },

  contactSection: {
    eyebrow: "Zaczynamy",
    title: "Napisz jedno zdanie. Reszta jest prosta.",
    lede: "Nie musisz wiedzieć, na czym chcesz grać ani ile masz czasu. Od tego jest pierwsza rozmowa.",
  },

  footer: {
    tagline: "Lekcje pianina, keyboardu i gitary dla dorosłych — stacjonarnie i online.",
    // Miejsce na przyszłe dokumenty. Na teraz puste: Umami jest bezcookie'owe,
    // strona nie zbiera danych, więc banner ani polityka nie są wymagane.
    legalLinks: [] as { label: string; href: string }[],
  },
} as const;

export type Site = typeof site;
