/**
 * 轻量 HTML 消毒：保留公告常用标签，去掉 script/事件处理器。
 * 用于公告 content（旧站存 HTML）。
 */
const ALLOWED = new Set([
  'A',
  'B',
  'STRONG',
  'I',
  'EM',
  'BR',
  'P',
  'DIV',
  'SPAN',
  'UL',
  'OL',
  'LI',
  'H1',
  'H2',
  'H3',
  'H4',
  'CODE',
  'PRE',
  'HR',
  'IMG',
  'U',
  'SMALL',
])

const ALLOWED_ATTR = new Set([
  'href',
  'target',
  'rel',
  'src',
  'alt',
  'title',
])

export type SanitizeOptions = {
  /** Strip class/style so host UI can restyle (e.g. mecha announcement). Default true for safety. */
  stripPresentation?: boolean
}

export function sanitizeHtml(html: string, opts: SanitizeOptions = {}): string {
  if (!html || typeof html !== 'string') return ''
  // 若看起来不是 HTML，当纯文本处理
  if (!/[<>]/.test(html)) {
    return escapeText(html)
  }

  const stripPresentation = opts.stripPresentation !== false

  const doc = new DOMParser().parseFromString(`<div id="root">${html}</div>`, 'text/html')
  const root = doc.getElementById('root')
  if (!root) return escapeText(html)

  walk(root, stripPresentation)
  return root.innerHTML
}

function walk(node: Node, stripPresentation: boolean) {
  const children = Array.from(node.childNodes)
  for (const child of children) {
    if (child.nodeType === Node.ELEMENT_NODE) {
      const el = child as HTMLElement
      const tag = el.tagName.toUpperCase()
      if (!ALLOWED.has(tag)) {
        // unwrap: keep children text
        while (el.firstChild) {
          node.insertBefore(el.firstChild, el)
        }
        node.removeChild(el)
        continue
      }
      // strip bad attrs — never keep style/class for announcement restyle
      for (const attr of Array.from(el.attributes)) {
        const name = attr.name.toLowerCase()
        if (name.startsWith('on')) {
          el.removeAttribute(attr.name)
          continue
        }
        if (stripPresentation && (name === 'style' || name === 'class' || name === 'id')) {
          el.removeAttribute(attr.name)
          continue
        }
        if (!ALLOWED_ATTR.has(name) && !(name === 'class' || name === 'style' || name === 'width' || name === 'height')) {
          el.removeAttribute(attr.name)
          continue
        }
        if (name === 'href' || name === 'src') {
          const v = attr.value.trim().toLowerCase()
          if (v.startsWith('javascript:') || v.startsWith('data:text/html')) {
            el.removeAttribute(attr.name)
          }
        }
      }
      if (tag === 'A') {
        el.setAttribute('rel', 'noopener noreferrer')
        if (!el.getAttribute('target')) el.setAttribute('target', '_blank')
      }
      walk(el, stripPresentation)
    }
  }
}

function escapeText(s: string) {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

/** 是否像 HTML 内容 */
export function looksLikeHtml(s: string) {
  return /<\/?[a-z][\s\S]*>/i.test(s || '')
}
