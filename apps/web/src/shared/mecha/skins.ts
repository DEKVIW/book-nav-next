/**
 * Card mecha skins — action crop in TOP-RIGHT, never affects card height.
 * Stable: siteId % length
 */

export interface MechaSkin {
  id: string
  accent: string
  accentDim: string
  silhouette: string
  /** soft wash behind text area only */
  wash: string
}

export const MECHA_SKINS: MechaSkin[] = [
  {
    id: 'prime',
    accent: '#5ef0ff',
    accentDim: 'rgba(94, 240, 255, 0.45)',
    silhouette: '/mecha/silhouettes/s0.svg',
    wash: 'rgba(40, 140, 200, 0.12)',
  },
  {
    id: 'scout',
    accent: '#ffc857',
    accentDim: 'rgba(255, 200, 87, 0.45)',
    silhouette: '/mecha/silhouettes/s1.svg',
    wash: 'rgba(200, 130, 30, 0.12)',
  },
  {
    id: 'aerial',
    accent: '#c9a6ff',
    accentDim: 'rgba(201, 166, 255, 0.45)',
    silhouette: '/mecha/silhouettes/s2.svg',
    wash: 'rgba(130, 90, 210, 0.12)',
  },
  {
    id: 'heavy',
    accent: '#8ec5ff',
    accentDim: 'rgba(142, 197, 255, 0.45)',
    silhouette: '/mecha/silhouettes/s3.svg',
    wash: 'rgba(70, 120, 190, 0.12)',
  },
  {
    id: 'engineer',
    accent: '#5dffc4',
    accentDim: 'rgba(93, 255, 196, 0.45)',
    silhouette: '/mecha/silhouettes/s4.svg',
    wash: 'rgba(30, 160, 120, 0.12)',
  },
  {
    id: 'shadow',
    accent: '#b8c6e0',
    accentDim: 'rgba(184, 198, 224, 0.45)',
    silhouette: '/mecha/silhouettes/s5.svg',
    wash: 'rgba(100, 120, 160, 0.12)',
  },
]

export function skinForId(id: number | string | undefined | null): MechaSkin {
  const n = typeof id === 'number' ? id : Number(id) || 0
  return MECHA_SKINS[Math.abs(Math.trunc(n)) % MECHA_SKINS.length]
}
