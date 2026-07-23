/** Team URL helpers: /team/{name-slug}-{4char-code} */

const SHORT_CODE_RE = /^[a-z0-9]{4}$/

export function nameSlug(name: string): string {
  const s = (name || '').trim()
  let out = ''
  let prevDash = false
  for (const ch of s) {
    const code = ch.codePointAt(0) || 0
    const isAsciiLetter = (code >= 65 && code <= 90) || (code >= 97 && code <= 122)
    const isDigit = code >= 48 && code <= 57
    const isCJKOrLetter = isAsciiLetter || isDigit || code > 127
    if (isCJKOrLetter) {
      out += isAsciiLetter && code <= 90 ? ch.toLowerCase() : ch
      prevDash = false
    } else if (ch === ' ' || ch === '-' || ch === '_') {
      if (out && !prevDash) {
        out += '-'
        prevDash = true
      }
    }
  }
  return out.replace(/^-+|-+$/g, '')
}

/** Build SPA path for a team. */
export function buildTeamPath(name: string, code: string): string {
  const slug = nameSlug(name)
  if (!slug) return `/team/${code}`
  return `/team/${slug}-${code}`
}

export function parseTeamPath(segment: string): { code: string; nameSlug: string } {
  const raw = decodeURIComponent((segment || '').trim())
  if (!raw) return { code: '', nameSlug: '' }
  const parts = raw.split('-')
  if (parts.length >= 2) {
    const last = parts[parts.length - 1]
    if (SHORT_CODE_RE.test(last)) {
      return { code: last, nameSlug: parts.slice(0, -1).join('-') }
    }
  }
  // bare short code or legacy long code (alias grace)
  return { code: raw, nameSlug: '' }
}

export function isShortCode(code: string): boolean {
  return SHORT_CODE_RE.test(code || '')
}
