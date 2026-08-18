import type { User } from '../api/types'

/**
 * A5: one identity rule for every surface that names the customer (the
 * header pill, the rail's profile card). Prefer the profile's name; fall
 * back to the email's local part — the same truthful fallback A1 shipped
 * before names existed.
 */
export function displayName(user: Pick<User, 'full_name' | 'email'>): string {
  return user.full_name.trim() || user.email.split('@')[0]
}

/**
 * The avatar's letters: initials of the first two words of the name
 * ("Anahit Sargsyan" → "AS"), or the first letter of the fallback.
 * toLocaleUpperCase with no locale argument follows the runtime's — fine
 * here, since Armenian and Cyrillic both uppercase unambiguously.
 */
export function initials(user: Pick<User, 'full_name' | 'email'>): string {
  const name = user.full_name.trim()
  if (name) {
    return name
      .split(/\s+/)
      .slice(0, 2)
      .map((w) => w[0])
      .join('')
      .toLocaleUpperCase()
  }
  return (user.email[0] ?? '?').toLocaleUpperCase()
}
