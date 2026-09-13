// Maps browser paths to neutral or persona-specific document titles for SPA navigation.

export const documentTitleByPath = (pathname: string): string => {
  if (pathname === "/learners" || pathname.startsWith("/learners/")) return "nutka — przestrzeń ucznia";
  if (pathname === "/teachers" || pathname.startsWith("/teachers/")) return "nutka — przestrzeń nauczyciela";
  return "nutka";
};

export function updateDocumentTitle(pathname: string): void {
  if (typeof document !== "undefined") document.title = documentTitleByPath(pathname);
}
