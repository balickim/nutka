/**
 * Źródło treści landingu oraz podstrony /cennik.
 *
 * Treści marketingowe są tutaj, a dane zależne od wdrożenia — w ./config.ts.
 *
 * Copy pisane jest w czasie teraźniejszym i formach bezosobowych. Polski czas przeszły
 * jest rodzajowy ("zaczynałeś" / "zaczynałaś"), a strona mówi do wszystkich odbiorców naraz.
 */

import { config } from "./config";

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

const teacherName = config.teacherName;
const city = config.contact.city;

export const site = {
  teacherName,

  meta: {
    title: `${teacherName} — lekcje gitary i keyboardu dla dorosłych · ${city}`,
    description: `Indywidualne lekcje gitary i keyboardu dla dorosłych i seniorów — ${city}. Zajęcia stacjonarne, 45 minut, bez egzaminów i ocen. Pierwsza lekcja próbna jest bezpłatna.`,
    ogImage: "og.png",
  },

  contact: {
    ...config.contact,
  },

  nav: [
    { label: "Lekcje", href: "/#lekcje" },
    { label: "O mnie", href: "/#o-mnie" },
    { label: "Cennik", href: "/cennik" },
    { label: "FAQ", href: "/#faq" },
  ],

  hero: {
    eyebrow: `Prywatne lekcje gitary i keyboardu · ${city}`,
    title: "Nauka gry na gitarze i keyboardzie",
    titleAccent: "dla dorosłych, od zera.",
    lede: "Indywidualne lekcje w mojej pracowni. Uczę dorosłych i seniorów, którzy chcą grać dla własnej przyjemności — bez egzaminów, ocen i presji, w swoim tempie.",
    // Fakty zamykają pytanie „czy to dla mnie i na jakich warunkach” bez przewijania strony.
    facts: [
      { label: "Dla kogo", value: "Dorośli i seniorzy, od zera lub po przerwie" },
      { label: "Forma", value: "Lekcje indywidualne, 45 minut" },
      { label: "Gdzie", value: `Stacjonarnie, ${city}` },
      { label: "Ile", value: `Od ${config.pricing.from} · lekcja próbna 0 zł` },
    ],
    ctaPrimary: "Umów bezpłatną lekcję próbną",
    ctaSecondary: "Zobacz, jak zacząć",
    photo: config.photos.hero,
  },

  objections: {
    eyebrow: "Trzy rzeczy, które zwykle zatrzymują",
    title: "To nie jest szkoła muzyczna.",
    items: [
      {
        claim: "„Jestem za stary, żeby zaczynać”",
        answer:
          "Dorosły uczeń ma przewagę, o której rzadko się mówi: wie, po co tu jest. Sam wybiera utwory i rozumie, co ćwiczy. Uczę dorosłych i seniorów — pierwszy kontakt z instrumentem na emeryturze to normalny start, a nie wyjątek.",
      },
      {
        claim: "„Nie mam talentu”",
        answer:
          "Gra na instrumencie to zestaw umiejętności, które da się rozłożyć na małe kroki i przejść po kolei. Najwięcej zależy od tego, ile ćwiczysz między lekcjami — talent przyspiesza początki nauki, a nie całą resztę.",
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
    title: "Gitara albo keyboard. Jeden powód: chcesz grać.",
    items: [
      {
        name: "Gitara klasyczna i akustyczna",
        symbol: "♫",
        description:
          "Akordy do śpiewania w gronie znajomych, palcówki, ulubione utwory — zaczynamy od tego, co chcesz zagrać.",
      },
      {
        name: "Keyboard",
        symbol: "♪",
        description:
          "Od pierwszego akordu po utwór, który chodzi Ci po głowie od lat. Klasyka, pop, muzyka filmowa — repertuar wybieramy razem. Do ćwiczeń w domu keyboard w zupełności wystarczy.",
      },
    ],
  },

  about: {
    eyebrow: "O mnie",
    title: "Uczę tak, żeby chciało się wrócić do instrumentu w kolejnym tygodniu.",
    paragraphs: [
      "Od wielu lat uczę gry na gitarze i keyboardzie — dzieci i dorosłych. Praca z dziećmi nauczyła mnie rozkładania trudnych rzeczy na najmniejsze możliwe kroki. Praca z dorosłymi — tego, że o powodzeniu decyduje nie talent, tylko to, czy lekcja daje frajdę.",
      "TODO: wykształcenie muzyczne, ukończone szkoły, doświadczenie sceniczne.",
      "TODO: zdanie osobiste — dlaczego uczysz i co najbardziej cieszy Cię w tej pracy.",
    ],
    portrait: config.photos.portrait,
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
        description: "Plan i materiały do ćwiczeń czekają na Twoim profilu, a terminy dopasowujemy do Twojego tygodnia.",
      },
    ],
  },

  app: {
    available: config.app.available,
    badge: "W przygotowaniu",
    eyebrow: "Aplikacja dla uczniów",
    title: "Wszystko z lekcji w jednym miejscu.",
    lede: "Po lekcji materiały czekają na Twoim profilu — nagrania, nuty i plan na najbliższy tydzień.",
    ctaLabel: "Zaloguj się do aplikacji",
    features: [
      { name: "Nagrania i nuty", description: "Wracasz do materiału z lekcji, kiedy tylko chcesz." },
      { name: "Zadania", description: "Wiesz dokładnie, co ćwiczyć w danym tygodniu." },
      { name: "Terminy", description: "Umawiasz i przekładasz zajęcia bez dzwonienia." },
    ],
  },

  testimonials: {
    // Zostaje wyłączone dopóki nie pojawią się prawdziwe wypowiedzi uczniów.
    // Zmyślone opinie z imionami wprowadzają klienta w błąd — sekcja włącza się jedną linią.
    enabled: config.testimonials.enabled,
    eyebrow: "Opinie",
    title: "Co mówią uczniowie.",
    items: config.testimonials.items,
  },

  pricing: {
    eyebrow: "Cennik",
    title: "Jasne stawki, bez ukrytych kosztów.",
    from: config.pricing.from,
    fromUnit: "za 45 minut",
    teaserNote: "Pierwsza lekcja próbna jest bezpłatna — sprawdzasz bez zobowiązań.",
    ctaLabel: "Zobacz pełny cennik",
    items: config.pricing.items,
    rules: config.pricing.rules,
  },

  faq: {
    eyebrow: "FAQ",
    title: "Pytania, które padają najczęściej.",
    items: [
      {
        question: "Gdzie odbywają się lekcje?",
        answer: `Lekcje odbywają się stacjonarnie w mojej pracowni: ${config.contact.address}, ${city}. Zajęcia prowadzę wyłącznie na miejscu — przy nauce gry wiele rzeczy, jak ustawienie ręki, poprawia się dopiero przy instrumencie.`,
      },
      {
        question: "Czy uczysz też dzieci?",
        answer:
          "Tak. Od lat uczę zarówno dzieci, jak i dorosłych — ta strona opisuje zajęcia dla dorosłych, ale jeśli szukasz lekcji dla dziecka, po prostu napisz.",
      },
      {
        question: "Nie mam instrumentu w domu. Co wtedy?",
        answer:
          "Na pierwszą lekcję instrument nie jest potrzebny, ale do ćwiczenia w domu już tak. Przy gitarze początkującym polecam klasyczną: nylonowe struny są łagodniejsze dla nieprzyzwyczajonych palców, a szerszy gryf wybacza więcej przy ustawianiu ręki. Przy klawiszach na start w zupełności wystarczy zwykły keyboard. Chętnie doradzę przed zakupem, żeby nie przepłacić.",
      },
      {
        question: "Czy muszę znać nuty?",
        answer:
          "Nie. Nut uczysz się w takim zakresie, w jakim są Ci realnie potrzebne do grania. Można też pracować na akordach i tabulaturach.",
      },
      {
        question: "Skąd biorę nuty do ćwiczenia?",
        answer:
          "Do każdego utworu przygotowuję własne opracowanie nut i akordów, dopasowane do poziomu konkretnego ucznia. Zamiast szukać wersji, którą da się zagrać, dostajesz zapis pisany pod Twoje możliwości — a przy kolejnych podejściach ten sam utwór wraca w trudniejszej wersji. Opracowanie razem z nagraniem z lekcji trafia na Twój profil w aplikacji.",
      },
      {
        question: "Jak często odbywają się lekcje?",
        answer:
          "Standardowo raz w tygodniu — w tym rytmie postępy są najbardziej widoczne. Jeśli Twój grafik na to nie pozwala, ustalimy inny.",
      },
      {
        question: "Co, jeśli muszę odwołać lekcję?",
        answer:
          "Wystarczy dać znać najpóźniej 24 godziny wcześniej — wtedy przekładamy albo odwołujemy zajęcia bez opłaty. W Planie Regularnym masz jedno bezpłatne przełożenie w miesiącu i dwa bezpłatne odwołania w okresie umowy. Szczegóły są w regulaminie.",
      },
    ],
  },

  contactSection: {
    eyebrow: "Zaczynamy",
    title: "Napisz jedno zdanie. Reszta jest prosta.",
    lede: "Nie musisz wiedzieć, na czym chcesz grać ani ile masz czasu. Od tego jest pierwsza rozmowa.",
  },

  footer: {
    tagline: `Indywidualne lekcje gitary i keyboardu dla dorosłych i seniorów. Stacjonarnie, ${city}.`,
    // Umami jest bezcookie'owe i strona nie zbiera danych, więc banner ani
    // polityka prywatności nie są wymagane. Regulamin zajęć jest tu jako PDF.
    legalLinks: [{ label: "Regulamin zajęć (PDF)", href: "/regulamin.pdf" }] as {
      label: string;
      href: string;
    }[],
  },
} as const;

export type Site = typeof site;
