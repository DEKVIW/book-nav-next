/**
 * Mecha HUD line icons — angular stroke style (shared portal + admin).
 * Paths are 24x24 viewBox.
 */

export type MechaIconName =
  | 'speedometer'
  | 'gear'
  | 'folder'
  | 'globe'
  | 'people'
  | 'server'
  | 'image'
  | 'archive'
  | 'house'
  | 'list'
  | 'chevron-left'
  | 'chevron-right'
  | 'box-arrow-right'
  | 'activity'
  | 'search'
  | 'plus'
  | 'close'
  | 'link'
  | 'shield'
  | 'bolt'
  | 'radar'

/** Angular mecha-flavored path data */
export const MECHA_ICON_PATHS: Record<MechaIconName, string> = {
  speedometer:
    'M12 3.5A8.5 8.5 0 1 0 20.5 12h-2A6.5 6.5 0 1 1 12 5.5V3.5zm1 5.2l3.8 3.8-1.4 1.4L11 10.5V6.7h2z',
  gear: 'M11 2h2l.6 2.2 2-.9 1.4 1.4-.9 2 2.2.6v2l-2.2.6.9 2-1.4 1.4-2-.9L13 20h-2l-.6-2.2-2 .9-1.4-1.4.9-2L5.7 13v-2l2.2-.6-.9-2L8.4 6.3l2 .9L11 2zm1 7a3 3 0 1 0 0 6 3 3 0 0 0 0-6z',
  folder:
    'M3 6.5L4.5 5h5l1.5 1.5H19.5L21 8v10.5L19.5 20h-15L3 18.5V6.5zm2 2v9h14v-8H11.2L9.7 8.5H5z',
  globe:
    'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zm0 2c1.4 1.6 2.2 3.7 2.4 6H9.6C9.8 8.7 10.6 6.6 12 5zm-4.5 8c.3 2.3 1.2 4.4 2.7 6H9.8A7 7 0 0 1 7.5 13zm9 0a7 7 0 0 1-2.3 6h.4c1.5-1.6 2.4-3.7 2.7-6h-.8zM9.6 11h4.8c-.2 2.3-1 4.4-2.4 6-1.4-1.6-2.2-3.7-2.4-6z',
  people:
    'M9 8a3 3 0 1 0 0-6 3 3 0 0 0 0 6zm6 1a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5zM3.5 20v-1.5C3.5 15.5 6 14 9 14s5.5 1.5 5.5 4.5V20h-11zm12.2-1.2c.4-1.8 1.9-3 4.3-3 2.2 0 3.5 1 3.5 3V20h-7.8v-1.2z',
  server:
    'M3 5.5L4.5 4h15L21 5.5v3L19.5 10h-15L3 8.5v-3zm0 7L4.5 11h15L21 12.5v3L19.5 17h-15L3 15.5v-3zm0 7L4.5 18h15L21 19.5V21H3v-1.5zM6 7h2v1H6V7zm0 7h2v1H6v-1zm0 7h2v1H6v-1z',
  image:
    'M3 6.5L4.5 5h15L21 6.5v11L19.5 19h-15L3 17.5v-11zm2 1.5v8.2l3.5-3.5 2.5 2.5 4-4L19 15.8V8H5zm3 2.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3z',
  archive:
    'M3 5.5L4.5 4h15L21 5.5V8H3V5.5zM4 9h16v10.5L18.5 21h-13L4 19.5V9zm5 3h6v2H9v-2z',
  house: 'M3 11.5L12 4l9 7.5V20h-6v-6H9v6H3v-8.5z',
  list: 'M4 6h16v2H4V6zm0 5h16v2H4v-2zm0 5h16v2H4v-2z',
  'chevron-left': 'M14.5 6L8.5 12l6 6 1.4-1.4L11.3 12l4.6-4.6L14.5 6z',
  'chevron-right': 'M9.5 6L15.5 12l-6 6-1.4-1.4 4.6-4.6-4.6-4.6L9.5 6z',
  'box-arrow-right':
    'M4 5.5L5.5 4H12v2H6v12h6v2H5.5L4 18.5v-13zM13 11h5.2l-1.6-1.6L18 8l4 4-4 4-1.4-1.4 1.6-1.6H13v-2z',
  activity: 'M3 13h3l2.5-7 3 14 2.5-9H21',
  search:
    'M10.5 4a6.5 6.5 0 1 1 0 13 6.5 6.5 0 0 1 0-13zm0 2a4.5 4.5 0 1 0 0 9 4.5 4.5 0 0 0 0-9zm6.2 9.1 3.2 3.2-1.4 1.4-3.2-3.2 1.4-1.4z',
  plus: 'M11 5h2v6h6v2h-6v6h-2v-6H5v-2h6V5z',
  close: 'M6.4 5L12 10.6 17.6 5 19 6.4 13.4 12 19 17.6 17.6 19 12 13.4 6.4 19 5 17.6 10.6 12 5 6.4 6.4 5z',
  link: 'M9 11h6v2H9v-2zm-2.5 1a3.5 3.5 0 0 1 3.5-3.5H12v2h-2a1.5 1.5 0 0 0 0 3h2v2h-2A3.5 3.5 0 0 1 6.5 12zm7-3.5H12v2h2a1.5 1.5 0 0 1 0 3h-2v2h2a3.5 3.5 0 1 0 0-7z',
  shield: 'M12 3l8 3v6c0 5-3.5 8.5-8 10-4.5-1.5-8-5-8-10V6l8-3zm0 2.2L6 7.1v4.9c0 3.6 2.4 6.4 6 7.7 3.6-1.3 6-4.1 6-7.7V7.1l-6-1.9z',
  bolt: 'M13 2L6 13h5l-1 9 8-12h-5l0-8z',
  radar:
    'M12 4a8 8 0 1 1 0 16 8 8 0 0 1 0-16zm0 2a6 6 0 1 0 0 12 6 6 0 0 0 0-12zm0 2a4 4 0 1 1 0 8 4 4 0 0 1 0-8zm-.8 3.2 4.5-2.2.9 1.8-4.5 2.2-.9-1.8z',
}
