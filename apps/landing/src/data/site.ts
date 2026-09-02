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

export const site = {
  teacherName,

  meta: {
    title: `${teacherName} — lekcje gitary i keyboardu dla dorosłych`,
    description:
      "Lekcje gitary i keyboardu dla dorosłych, którzy chcą grać dla własnej przyjemności. Bez egzaminów, ocen i presji, stacjonarnie i online. Umów lekcję próbną.",
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
    eyebrow: "Gitara i keyboard · stacjonarnie i online",
    title: "To nie jest szkoła muzyczna.",
    titleAccent: "Tu grasz dla siebie.",
    lede: "Bez egzaminów, ocen i presji. Uczę dorosłych, którzy chcą grać dla własnej przyjemności — w swoim tempie i na swoich warunkach.",
    ctaPrimary: "Umów lekcję próbną",
    ctaSecondary: "Zobacz, jak zacząć",
    photo: config.photos.hero,
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
    modes: ["Stacjonarnie", "Online", "Dla początkujących", "Dla powracających"],
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
        description: "Dostajesz plan i materiały do ćwiczeń, a terminy dopasowujemy do Twojego tygodnia.",
      },
    ],
  },

  app: {
    available: config.app.available,
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
    enabled: config.testimonials.enabled,
    eyebrow: "Opinie",
    title: "Co mówią uczniowie.",
    items: config.testimonials.items,
  },

  pricing: {
    eyebrow: "Cennik",
    title: "Jasne stawki, bez ukrytych kosztów.",
    from: config.pricing.from,
    fromUnit: "za 60 minut",
    teaserNote: "Lekcja próbna jest krótsza i tańsza — sprawdzasz bez zobowiązań.",
    ctaLabel: "Zobacz pełny cennik",
    items: config.pricing.items,
    rules: config.pricing.rules,
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
          "Na pierwszą lekcję instrument nie jest potrzebny, ale do ćwiczenia w domu już tak. Przy gitarze początkującym polecam klasyczną: nylonowe struny są łagodniejsze dla nieprzyzwyczajonych palców, a szerszy gryf wybacza więcej przy ustawianiu ręki. Przy klawiszach na start w zupełności wystarczy zwykły keyboard. Chętnie doradzę przed zakupem, żeby nie przepłacić.",
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
        question: "Skąd biorę nuty do ćwiczenia?",
        answer:
          "Do każdego utworu przygotowuję własne opracowanie nut i akordów, dopasowane do poziomu konkretnego ucznia. Zamiast szukać wersji, którą da się zagrać, dostajesz zapis pisany pod Twoje możliwości — a przy kolejnych podejściach ten sam utwór wraca w trudniejszej wersji.",
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
    tagline: "Lekcje gitary i keyboardu dla dorosłych — stacjonarnie i online.",
    // Miejsce na przyszłe dokumenty. Na teraz puste: Umami jest bezcookie'owe,
    // strona nie zbiera danych, więc banner ani polityka nie są wymagane.
    legalLinks: [] as { label: string; href: string }[],
  },
} as const;

export type Site = typeof site;
