/**
 * Wjazd sekcji przy scrollu. Bez bibliotek.
 *
 * Stan ukryty żyje wyłącznie w CSS pod selektorem `.js .reveal`, więc bez JavaScriptu
 * treść jest normalnie widoczna. Przy `prefers-reduced-motion: reduce` CSS w ogóle nie ukrywa
 * sekcji, ale i tak wychodzimy wcześniej, żeby nie zawieszać observera bez potrzeby.
 */
const prefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
const sections = document.querySelectorAll<HTMLElement>(".reveal");

if (prefersReducedMotion || !("IntersectionObserver" in window)) {
  sections.forEach((section) => section.classList.add("is-visible"));
} else {
  const observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (!entry.isIntersecting) continue;
        entry.target.classList.add("is-visible");
        observer.unobserve(entry.target);
      }
    },
    { rootMargin: "0px 0px -12% 0px", threshold: 0.05 },
  );

  sections.forEach((section) => observer.observe(section));
}
