/**
 * Joins class names, dropping anything falsy so callers can write
 * `cx('base', isActive && 'active')` without littering the output with
 * "false" or empty strings.
 *
 * The npm ecosystem has `clsx` for this, but it is nine lines of work and
 * the project's rule is no dependency without a reason (docs/RULES.md).
 *
 * C++ note: the `...parts` rest parameter is a variadic like
 * `template <typename... Ts> void f(Ts...)`, except everything is one type
 * here and the values arrive as a real array rather than a parameter pack —
 * so `filter`/`join` work on it directly, no recursion or fold expression.
 */
export function cx(...parts: (string | false | null | undefined)[]): string {
  return parts.filter(Boolean).join(' ')
}
