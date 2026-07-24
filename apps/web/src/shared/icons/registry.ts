/**
 * Lucide icon registry — tree-shakeable whitelist + legacy name aliases.
 * Storage convention: kebab-case (folder, globe, layers, …)
 */
import type { FunctionalComponent } from 'vue'
import {
  Activity,
  Archive,
  BarChart3,
  Bell,
  Book,
  Bookmark,
  Box,
  Boxes,
  Camera,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Cloud,
  Code2,
  Compass,
  Cpu,
  CreditCard,
  Download,
  ExternalLink,
  Film,
  Folder,
  FolderOpen,
  Gamepad2,
  Gauge,
  Gift,
  Github,
  Globe,
  Heart,
  Hexagon,
  Home,
  Image,
  Info,
  Layers,
  LayoutDashboard,
  Link2,
  List,
  LogOut,
  Menu,
  Moon,
  Music,
  Newspaper,
  Package,
  Puzzle,
  Radar,
  Search,
  Server,
  Settings,
  Shield,
  ShoppingBag,
  Sparkles,
  Star,
  Tags,
  Ticket,
  Users,
  Wrench,
  X,
  Zap,
  type LucideProps,
} from 'lucide-vue-next'

export type LucideIcon = FunctionalComponent<LucideProps>

/** kebab-case keys used in DB / UI */
export const ICON_REGISTRY: Record<string, LucideIcon> = {
  activity: Activity,
  archive: Archive,
  'bar-chart': BarChart3,
  'bar-chart-3': BarChart3,
  bell: Bell,
  book: Book,
  bookmark: Bookmark,
  box: Box,
  boxes: Boxes,
  camera: Camera,
  'chevron-down': ChevronDown,
  'chevron-left': ChevronLeft,
  'chevron-right': ChevronRight,
  'chevrons-left': ChevronsLeft,
  'chevrons-right': ChevronsRight,
  cloud: Cloud,
  code: Code2,
  'code-2': Code2,
  compass: Compass,
  cpu: Cpu,
  'credit-card': CreditCard,
  download: Download,
  'external-link': ExternalLink,
  film: Film,
  folder: Folder,
  'folder-open': FolderOpen,
  gamepad: Gamepad2,
  'gamepad-2': Gamepad2,
  gauge: Gauge,
  gift: Gift,
  github: Github,
  globe: Globe,
  heart: Heart,
  hexagon: Hexagon,
  home: Home,
  house: Home,
  image: Image,
  info: Info,
  layers: Layers,
  collection: Layers,
  'layout-dashboard': LayoutDashboard,
  dashboard: LayoutDashboard,
  link: Link2,
  'link-2': Link2,
  list: List,
  logout: LogOut,
  'log-out': LogOut,
  menu: Menu,
  moon: Moon,
  music: Music,
  newspaper: Newspaper,
  package: Package,
  puzzle: Puzzle,
  radar: Radar,
  search: Search,
  server: Server,
  settings: Settings,
  gear: Settings,
  shield: Shield,
  'shopping-bag': ShoppingBag,
  sparkles: Sparkles,
  star: Star,
  tags: Tags,
  tag: Tags,
  ticket: Ticket,
  users: Users,
  people: Users,
  wrench: Wrench,
  tools: Wrench,
  tool: Wrench,
  x: X,
  close: X,
  zap: Zap,
  bolt: Zap,
  lightning: Zap,
  speedometer: Gauge,
}

/** Curated set for category icon picker (label for search) */
export const CATEGORY_ICON_OPTIONS: { name: string; label: string }[] = [
  { name: 'folder', label: '文件夹' },
  { name: 'layers', label: '合集' },
  { name: 'globe', label: '地球' },
  { name: 'film', label: '影视' },
  { name: 'cloud', label: '云' },
  { name: 'zap', label: '闪电' },
  { name: 'book', label: '书本' },
  { name: 'bookmark', label: '书签' },
  { name: 'puzzle', label: '拼图' },
  { name: 'tags', label: '标签' },
  { name: 'camera', label: '相机' },
  { name: 'wrench', label: '工具' },
  { name: 'image', label: '图片' },
  { name: 'gauge', label: '仪表' },
  { name: 'server', label: '服务器' },
  { name: 'shield', label: '盾牌' },
  { name: 'sparkles', label: 'AI / 星芒' },
  { name: 'code', label: '代码' },
  { name: 'download', label: '下载' },
  { name: 'compass', label: '指南针' },
  { name: 'box', label: '盒子' },
  { name: 'boxes', label: '多盒' },
  { name: 'package', label: '包裹' },
  { name: 'link', label: '链接' },
  { name: 'star', label: '星标' },
  { name: 'heart', label: '收藏' },
  { name: 'music', label: '音乐' },
  { name: 'gamepad', label: '游戏' },
  { name: 'newspaper', label: '资讯' },
  { name: 'shopping-bag', label: '购物' },
  { name: 'users', label: '用户' },
  { name: 'home', label: '主页' },
  { name: 'search', label: '搜索' },
  { name: 'radar', label: '雷达' },
  { name: 'cpu', label: '芯片' },
  { name: 'hexagon', label: '六边形' },
  { name: 'archive', label: '归档' },
  { name: 'bell', label: '通知' },
  { name: 'gift', label: '礼品' },
  { name: 'ticket', label: '票据' },
  { name: 'credit-card', label: '卡片' },
  { name: 'bar-chart', label: '图表' },
  { name: 'activity', label: '活动' },
  { name: 'settings', label: '设置' },
]

const LEGACY_ALIASES: Record<string, string> = {
  collection: 'layers',
  lightning: 'zap',
  bolt: 'zap',
  gear: 'settings',
  people: 'users',
  house: 'home',
  tools: 'wrench',
  tool: 'wrench',
  speedometer: 'gauge',
  'speedometer2': 'gauge',
  bi: 'folder',
  fa: 'folder',
  'fa-folder': 'folder',
  'fa-globe': 'globe',
  'fa-film': 'film',
  'fa-cloud': 'cloud',
  'fa-bolt': 'zap',
  'fa-book': 'book',
  'fa-bookmark': 'bookmark',
  'fa-puzzle-piece': 'puzzle',
  'fa-tags': 'tags',
  'fa-camera': 'camera',
  'fa-tools': 'wrench',
  'fa-image': 'image',
  'fa-cog': 'settings',
  'bi-folder': 'folder',
  'bi-globe': 'globe',
  'bi-film': 'film',
  'bi-cloud': 'cloud',
  'bi-lightning': 'zap',
  'bi-book': 'book',
  'bi-bookmark': 'bookmark',
  'bi-puzzle': 'puzzle',
  'bi-tags': 'tags',
  'bi-camera': 'camera',
  'bi-tools': 'wrench',
  'bi-image': 'image',
  'bi-collection': 'layers',
  'bi-speedometer2': 'gauge',
  'bi-gear': 'settings',
  'bi-people': 'users',
  'box-arrow-right': 'log-out',
  'chevron-left': 'chevron-left',
  'chevron-right': 'chevron-right',
}

function toKebab(raw: string): string {
  return raw
    .trim()
    .replace(/^fa-solid\s+/i, '')
    .replace(/^fas\s+/i, '')
    .replace(/^fa\s+/i, '')
    .replace(/^bi\s+/i, '')
    .replace(/([a-z])([A-Z])/g, '$1-$2')
    .replace(/[\s_]+/g, '-')
    .replace(/-+/g, '-')
    .toLowerCase()
}

/** Resolve any stored icon string to a registry key */
export function resolveIconName(icon?: string | null, fallback = 'folder'): string {
  if (!icon || !String(icon).trim()) return fallback
  let key = toKebab(String(icon))
  if (LEGACY_ALIASES[key]) key = LEGACY_ALIASES[key]
  // strip one fa-/bi- prefix
  if (key.startsWith('fa-') || key.startsWith('bi-')) {
    const stripped = key.slice(3)
    key = LEGACY_ALIASES[key] || LEGACY_ALIASES[stripped] || stripped
  }
  if (ICON_REGISTRY[key]) return key
  return fallback
}

export function getLucideIcon(icon?: string | null, fallback = 'folder'): LucideIcon {
  const key = resolveIconName(icon, fallback)
  return ICON_REGISTRY[key] || ICON_REGISTRY[fallback] || Folder
}

/** Cycle fallback by id when icon missing */
export function iconForCategory(icon?: string | null, id?: number | null): string {
  if (icon && String(icon).trim()) return resolveIconName(icon)
  const cycle = [
    'folder',
    'globe',
    'zap',
    'film',
    'cloud',
    'book',
    'wrench',
    'image',
    'radar',
    'shield',
    'sparkles',
    'code',
  ]
  const n = typeof id === 'number' ? Math.abs(id) : 0
  return cycle[n % cycle.length]
}
