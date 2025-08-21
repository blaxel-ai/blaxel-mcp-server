/**
 * Converts a string to a URL-safe slug format
 * - Converts to lowercase
 * - Replaces spaces and underscores with hyphens
 * - Removes special characters
 * - Removes consecutive hyphens
 * - Trims hyphens from start and end
 */
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/[\s_]+/g, '-')           // Replace spaces and underscores with hyphens
    .replace(/[^\w\-]+/g, '')          // Remove all non-word chars except hyphens
    .replace(/\-\-+/g, '-')            // Replace multiple hyphens with single hyphen
    .replace(/^-+/, '')                // Trim hyphens from start
    .replace(/-+$/, '');               // Trim hyphens from end
}
