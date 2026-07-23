/** Admin icons — re-export mecha HUD set for consistency */
import { MECHA_ICON_PATHS, type MechaIconName } from '@/shared/mecha/icons'

export type IconName = MechaIconName | 'envelope' | 'link-break' | 'journal'

export const ICON_PATHS: Record<string, string> = {
  ...MECHA_ICON_PATHS,
  envelope: MECHA_ICON_PATHS.people,
  'link-break': MECHA_ICON_PATHS.link,
  journal: MECHA_ICON_PATHS.list,
}
