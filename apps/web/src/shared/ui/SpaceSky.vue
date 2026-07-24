<script setup lang="ts">
/**
 * Canvas starfield — shared portal background / warp transition.
 *
 * intensity:
 *  - full: dense stars, bright nebulae, meteors (goto page)
 *  - ambient: lighter, slower, content-friendly (portal shell)
 */
import { onMounted, onUnmounted, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    intensity?: 'full' | 'ambient'
  }>(),
  { intensity: 'ambient' },
)

const canvasRef = ref<HTMLCanvasElement | null>(null)
let stop: (() => void) | null = null

type Star = { x: number; y: number; z: number; r: number; base: number; phase: number }
type Meteor = { x: number; y: number; vx: number; vy: number; life: number; max: number }

type Cfg = {
  areaDiv: number
  minStars: number
  maxStars: number
  brightChance: number
  drift: number
  meteorChance: number
  nebulaAlpha: number
  vignette: number
  clearBg: boolean
  starAlpha: number
  extraNebula: boolean
}

function preset(intensity: 'full' | 'ambient'): Cfg {
  if (intensity === 'full') {
    return {
      areaDiv: 1400,
      minStars: 450,
      maxStars: 1400,
      brightChance: 0.12,
      drift: 0.35,
      meteorChance: 0.008,
      nebulaAlpha: 1,
      vignette: 0.28,
      clearBg: true,
      starAlpha: 1,
      extraNebula: true,
    }
  }
  return {
    areaDiv: 3200,
    minStars: 180,
    maxStars: 480,
    brightChance: 0.06,
    drift: 0.12,
    meteorChance: 0.002,
    nebulaAlpha: 0.55,
    vignette: 0.38,
    clearBg: false,
    starAlpha: 0.88,
    extraNebula: false,
  }
}

function startSky(intensity: 'full' | 'ambient') {
  const canvas = canvasRef.value
  if (!canvas) return () => {}

  const reduce =
    typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const cfg = preset(intensity)
  const ctx = canvas.getContext('2d', { alpha: !cfg.clearBg })
  if (!ctx) return () => {}

  let w = 0
  let h = 0
  let dpr = 1
  let stars: Star[] = []
  let meteors: Meteor[] = []
  const t0 = performance.now()
  let running = true
  let raf = 0

  function resize() {
    dpr = Math.min(window.devicePixelRatio || 1, 2)
    w = window.innerWidth
    h = window.innerHeight
    canvas.width = Math.floor(w * dpr)
    canvas.height = Math.floor(h * dpr)
    canvas.style.width = `${w}px`
    canvas.style.height = `${h}px`
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    seedStars()
  }

  function seedStars() {
    const n = Math.min(cfg.maxStars, Math.max(cfg.minStars, Math.floor((w * h) / cfg.areaDiv)))
    stars = []
    for (let i = 0; i < n; i++) {
      const z = Math.random()
      const bright = Math.random() < cfg.brightChance
      stars.push({
        x: Math.random() * w,
        y: Math.random() * h,
        z: bright ? 0.85 + Math.random() * 0.15 : z,
        r: bright ? 1.6 + Math.random() * 1.4 : 0.7 + z * 1.8 + Math.random() * 0.6,
        base: bright ? 0.8 + Math.random() * 0.2 : 0.4 + z * 0.45,
        phase: Math.random() * Math.PI * 2,
      })
    }
  }

  function spawnMeteor() {
    if (meteors.length > 2) return
    meteors.push({
      x: Math.random() * w * 0.7,
      y: Math.random() * h * 0.45,
      vx: 5 + Math.random() * 7,
      vy: 2.5 + Math.random() * 3.5,
      life: 0,
      max: 30 + Math.random() * 25,
    })
  }

  function drawNebula(cx: number, cy: number, radius: number, color: string, drift: number) {
    const x = cx + Math.sin(drift) * 30
    const y = cy + Math.cos(drift * 0.8) * 20
    const grd = ctx.createRadialGradient(x, y, 0, x, y, radius)
    grd.addColorStop(0, color)
    grd.addColorStop(1, 'rgba(0,0,0,0)')
    ctx.fillStyle = grd
    ctx.fillRect(x - radius, y - radius, radius * 2, radius * 2)
  }

  function frame(now: number) {
    if (!running) return
    const t = (now - t0) / 1000
    const aMul = cfg.nebulaAlpha

    if (cfg.clearBg) {
      const g = ctx.createLinearGradient(0, 0, w * 0.3, h)
      g.addColorStop(0, '#0a1830')
      g.addColorStop(0.4, '#0c1a38')
      g.addColorStop(0.75, '#081428')
      g.addColorStop(1, '#060e20')
      ctx.fillStyle = g
      ctx.fillRect(0, 0, w, h)
    } else {
      ctx.clearRect(0, 0, w, h)
    }

    drawNebula(w * 0.2, h * 0.25, Math.min(w, h) * 0.55, `rgba(50,130,255,${0.32 * aMul})`, t * 0.02)
    drawNebula(w * 0.82, h * 0.58, Math.min(w, h) * 0.48, `rgba(140,70,220,${0.26 * aMul})`, -t * 0.015)
    drawNebula(w * 0.55, h * 0.12, Math.min(w, h) * 0.38, `rgba(40,200,240,${0.16 * aMul})`, t * 0.01)
    if (cfg.extraNebula) {
      drawNebula(w * 0.45, h * 0.78, Math.min(w, h) * 0.36, `rgba(30,90,200,${0.18 * aMul})`, t * 0.012)
    }

    for (const s of stars) {
      if (!reduce) {
        s.x += (0.12 + s.z * 0.45) * cfg.drift
        s.y += (0.04 + s.z * 0.16) * cfg.drift
        if (s.x > w + 4) s.x = -4
        if (s.y > h + 4) s.y = -4
      }
      const tw = reduce ? 1 : 0.55 + 0.45 * Math.sin(t * (1.1 + s.z) + s.phase)
      const a = Math.min(1, s.base * tw * cfg.starAlpha)
      const r = s.r * (0.85 + 0.25 * tw)

      if (s.z > 0.55) {
        const grd = ctx.createRadialGradient(s.x, s.y, 0, s.x, s.y, r * 3.5)
        grd.addColorStop(0, `rgba(180,240,255,${0.18 * a})`)
        grd.addColorStop(1, 'rgba(180,240,255,0)')
        ctx.fillStyle = grd
        ctx.beginPath()
        ctx.arc(s.x, s.y, r * 3.5, 0, Math.PI * 2)
        ctx.fill()
      }

      ctx.fillStyle = s.z > 0.7 ? `rgba(220,250,255,${a})` : `rgba(200,230,255,${a})`
      ctx.beginPath()
      ctx.arc(s.x, s.y, r, 0, Math.PI * 2)
      ctx.fill()
    }

    if (!reduce && Math.random() < cfg.meteorChance) spawnMeteor()
    meteors = meteors.filter((m) => m.life < m.max)
    for (const m of meteors) {
      m.life++
      m.x += m.vx
      m.y += m.vy
      const fade = 1 - m.life / m.max
      const len = 36 + m.vx * 3.5
      const ang = Math.atan2(m.vy, m.vx)
      ctx.strokeStyle = `rgba(200,240,255,${0.8 * fade})`
      ctx.lineWidth = 1.4
      ctx.beginPath()
      ctx.moveTo(m.x, m.y)
      ctx.lineTo(m.x - Math.cos(ang) * len, m.y - Math.sin(ang) * len)
      ctx.stroke()
    }

    if (cfg.vignette > 0) {
      const vg = ctx.createRadialGradient(w / 2, h * 0.42, w * 0.18, w / 2, h * 0.42, w * 0.82)
      vg.addColorStop(0, 'rgba(2,6,14,0)')
      vg.addColorStop(1, `rgba(2,6,14,${cfg.vignette})`)
      ctx.fillStyle = vg
      ctx.fillRect(0, 0, w, h)
    }

    raf = requestAnimationFrame(frame)
  }

  resize()
  window.addEventListener('resize', resize)
  raf = requestAnimationFrame(frame)

  return () => {
    running = false
    cancelAnimationFrame(raf)
    window.removeEventListener('resize', resize)
  }
}

function boot() {
  stop?.()
  stop = startSky(props.intensity)
}

onMounted(boot)
watch(() => props.intensity, boot)
onUnmounted(() => {
  stop?.()
  stop = null
})
</script>

<template>
  <canvas ref="canvasRef" class="space-sky" aria-hidden="true" />
</template>

<style scoped>
.space-sky {
  position: fixed;
  inset: 0;
  width: 100%;
  height: 100%;
  z-index: 0;
  display: block;
  pointer-events: none;
}
</style>
