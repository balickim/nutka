export function sitePath(path: string): string {
  if (!path.startsWith("/")) return path;

  const baseUrl = import.meta.env.BASE_URL.replace(/\/?$/, "/");
  return `${baseUrl}${path.slice(1)}`;
}
