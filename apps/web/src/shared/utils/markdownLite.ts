/**
 * Tiny Markdown subset for announcement content (no dependency).
 * Supports: **bold**, *italic*, `code`, [label](url), paragraphs, line breaks.
 * Output is structural tags only — presentation is owned by host CSS.
 */

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function inline(md: string): string {
  let s = escapeHtml(md)
  // links [text](url)
  s = s.replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g, (_m, label, url) => {
    const safe = String(url).replace(/"/g, '')
    return `<a href="${safe}" target="_blank" rel="noopener noreferrer">${label}</a>`
  })
  // bold **text** or __text__
  s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
  s = s.replace(/__([^_]+)__/g, '<strong>$1</strong>')
  // italic *text* (avoid bold leftovers)
  s = s.replace(/(^|[^*])\*([^*\n]+)\*(?!\*)/g, '$1<em>$2</em>')
  // inline code
  s = s.replace(/`([^`]+)`/g, '<code>$1</code>')
  return s
}

/** Convert a light MD string to safe HTML (structure only). */
export function markdownToHtml(md: string): string {
  if (!md || typeof md !== 'string') return ''
  const text = md.replace(/\r\n/g, '\n').trim()
  if (!text) return ''

  const blocks = text.split(/\n{2,}/)
  const parts: string[] = []
  for (const block of blocks) {
    const lines = block.split('\n')
    // unordered list
    if (lines.every((l) => /^\s*[-*]\s+/.test(l) || l.trim() === '')) {
      const items = lines
        .filter((l) => l.trim())
        .map((l) => `<li>${inline(l.replace(/^\s*[-*]\s+/, ''))}</li>`)
        .join('')
      parts.push(`<ul>${items}</ul>`)
      continue
    }
    // heading # ## ###
    const hm = /^(#{1,3})\s+(.+)$/.exec(lines[0].trim())
    if (hm && lines.length === 1) {
      const level = hm[1].length
      parts.push(`<h${level}>${inline(hm[2])}</h${level}>`)
      continue
    }
    parts.push(`<p>${lines.map((l) => inline(l)).join('<br>')}</p>`)
  }
  return parts.join('')
}

/** True if content looks like Markdown rather than HTML. */
export function looksLikeMarkdown(s: string): boolean {
  if (!s || /<\/?[a-z][\s\S]*>/i.test(s)) return false
  return /(\*\*[^*]+\*\*|__[^_]+__|`[^`]+`|\[[^\]]+\]\(https?:\/\/|^\s*#{1,3}\s+\S|^\s*[-*]\s+\S)/m.test(
    s,
  )
}
