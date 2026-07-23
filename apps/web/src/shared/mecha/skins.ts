/**
 * Mecha card skins — Transformers-inspired ORIGINAL variants.
 * Stable pick: siteId % skins.length
 */

export interface MechaSkin {
  id: string
  /** CSS accent for border / glow */
  accent: string
  accentDim: string
  /** silhouette asset under /mecha/silhouettes/ */
  silhouette: string
  /** soft panel tint */
  tint: string
}

export const MECHA_SKINS: MechaSkin[] = [
  {
    id: 'prime',
    accent: '#3de7ff',
    accentDim: 'rgba(61, 231, 255, 0.35)',
    silhouette: '/mecha/silhouettes/s0.svg',
    tint: 'rgba(40, 120, 180, 0.08)',
  },
  {
    id: 'scout',
    accent: '#ffb020',
    accentDim: 'rgba(255, 176, 32, 0.35)',
    silhouette: '/mecha/silhouettes/s1.svg',
    tint: 'rgba(180, 110, 20, 0.08)',
  },
  {
    id: 'aerial',
    accent: '#b388ff',
    accentDim: 'rgba(179, 136, 255, 0.35)',
    silhouette: '/mecha/silhouettes/s2.svg',
    tint: 'rgba(120, 80, 200, 0.08)',
  },
  {
    id: 'heavy',
    accent: '#7eb6ff',
    accentDim: 'rgba(126, 182, 255, 0.35)',
    silhouette: '/mecha/silhouettes/s3.svg',
    tint: 'rgba(60, 100, 160, 0.1)',
  },
  {
    id: 'engineer',
    accent: '#3dffb5',
    accentDim: 'rgba(61, 255, 181, 0.35)',
    silhouette: '/mecha/silhouettes/s4.svg',
    tint: 'rgba(30, 140, 100, 0.08)',
  },
  {
    id: 'shadow',
    accent: '#9aa8c7',
    accentDim: 'rgba(154, 168, 199, 0.35)',
    silhouette: '/mecha/silhouettes/s5.svg',
    tint: 'rgba(80, 90, 120, 0.1)',
  },
]

export function skinForId(id: number | string | undefined | null): MechaSkin {
  const n = typeof id === 'number' ? id : Number(id) || 0
  const idx = Math.abs(Math.trunc(n)) % MECHA_SKINS.length
  return MECHA_SKINS[idx]
}
