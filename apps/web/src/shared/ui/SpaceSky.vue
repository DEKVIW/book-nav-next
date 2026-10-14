<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'

type Theme = 'dark' | 'light'

const props = withDefaults(
  defineProps<{
    intensity?: 'full' | 'ambient'
    theme?: Theme
  }>(),
  { intensity: 'ambient', theme: 'dark' },
)

const canvasRef = ref<HTMLCanvasElement | null>(null)
let stop: (() => void) | null = null

type Star = { x: number; y: number; z: number; r: number; base: number; phase: number }
type Meteor = { x: number; y: number; vx: number; vy: number; life: number; max: number }
type CloudPuff = { x: number; y: number; rx: number; ry: number; alpha: number }
type Cloud = {
  x: number
  y: number
  width: number
  height: number
  alpha: number
  speed: number
  phase: number
  depth: number
  puffs: CloudPuff[]
}
type Dust = { x: number; y: number; radius: number; alpha: number; phase: number }

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

function startSky(intensity: 'full' | 'ambient', theme: Theme) {
  const canvasRefValue = canvasRef.value
  if (!canvasRefValue) return () => {}
  const canvas: HTMLCanvasElement = canvasRefValue

  const reduce =
    typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const cfg = preset(intensity)
  const context = canvas.getContext('2d', { alpha: true })
  if (!context) return () => {}
  const ctx: CanvasRenderingContext2D = context

  let w = 0
  let h = 0
  let dpr = 1
  let stars: Star[] = []
  let meteors: Meteor[] = []
  let clouds: Cloud[] = []
  let dust: Dust[] = []
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
    if (theme === 'dark') seedStars()
    else seedDaylight()
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

  function seedDaylight() {
    const cloudCount = intensity === 'full' ? 10 : 8
    clouds = Array.from({ length: cloudCount }, (_, index) => {
      const depth = 0.35 + Math.random() * 0.65
      const puffCount = 4 + Math.floor(Math.random() * 4)
      const puffs = Array.from({ length: puffCount }, (_, puffIndex) => ({
        x: -0.4 + (puffIndex / Math.max(1, puffCount - 1)) * 0.8 + (Math.random() - 0.5) * 0.14,
        y: -0.01 - Math.random() * 0.16,
        rx: 0.09 + Math.random() * 0.1,
        ry: 0.13 + Math.random() * 0.18,
        alpha: 0.72 + Math.random() * 0.28,
      }))
      return {
        x: Math.random() * w,
        y: h * (0.08 + Math.pow(Math.random(), 1.25) * 0.7),
        width: Math.max(220, w * (0.16 + depth * 0.26)),
        height: Math.max(44, h * (0.045 + depth * 0.065)),
        alpha: 0.12 + depth * 0.16,
        speed: 0.12 + depth * 0.22,
        phase: index * 1.7 + Math.random() * 2,
        depth,
        puffs,
      }
    }).sort((left, right) => left.depth - right.depth)
    const dustCount = intensity === 'full' ? 110 : 64
    dust = Array.from({ length: dustCount }, () => ({
      x: Math.random() * w,
      y: Math.random() * h,
      radius: 0.5 + Math.random() * 1.7,
      alpha: 0.12 + Math.random() * 0.18,
      phase: Math.random() * Math.PI * 2,
    }))
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

  function traceCloud(cloud: Cloud, x: number, y: number) {
    const points = [
      { x: -0.52, y: 0.18 },
      { x: -0.5, y: -0.01 },
      ...cloud.puffs.flatMap((puff) => [
        { x: puff.x - puff.rx, y: puff.y + puff.ry * 0.2 },
        { x: puff.x, y: puff.y - puff.ry },
        { x: puff.x + puff.rx, y: puff.y + puff.ry * 0.14 },
      ]),
      { x: 0.5, y: -0.01 },
      { x: 0.53, y: 0.2 },
      { x: 0.48, y: 0.36 },
      { x: 0.2, y: 0.43 },
      { x: -0.18, y: 0.4 },
      { x: -0.46, y: 0.34 },
    ]
    const scaled = points.map((point) => ({
      x: x + point.x * cloud.width,
      y: y + point.y * cloud.height,
    }))
    ctx.beginPath()
    ctx.moveTo(scaled[0].x, scaled[0].y)
    for (let index = 1; index <= scaled.length; index++) {
      const current = scaled[index % scaled.length]
      const next = scaled[(index + 1) % scaled.length]
      const midpoint = { x: (current.x + next.x) / 2, y: (current.y + next.y) / 2 }
      ctx.quadraticCurveTo(current.x, current.y, midpoint.x, midpoint.y)
    }
    ctx.closePath()
  }

  function drawCloud(cloud: Cloud, x: number, y: number) {
    traceCloud(cloud, x, y)
    ctx.save()
    ctx.globalAlpha = cloud.alpha * 0.26
    ctx.filter = `blur(${Math.max(5, cloud.height * 0.14)}px)`
    ctx.fillStyle = 'rgba(98, 136, 145, 0.8)'
    ctx.shadowColor = 'rgba(72, 108, 120, 0.18)'
    ctx.shadowBlur = 18
    ctx.shadowOffsetY = cloud.height * 0.18
    ctx.fill()
    ctx.restore()

    ctx.save()
    traceCloud(cloud, x, y)
    ctx.clip()
    const cloudGradient = ctx.createLinearGradient(0, y - cloud.height, 0, y + cloud.height)
    cloudGradient.addColorStop(0, `rgba(255,255,255,${cloud.alpha * 1.32})`)
    cloudGradient.addColorStop(0.42, `rgba(241,248,246,${cloud.alpha * 0.96})`)
    cloudGradient.addColorStop(0.74, `rgba(208,228,226,${cloud.alpha * 0.76})`)
    cloudGradient.addColorStop(1, `rgba(136,173,179,${cloud.alpha * 0.48})`)
    ctx.fillStyle = cloudGradient
    ctx.fillRect(x - cloud.width * 0.6, y - cloud.height * 0.7, cloud.width * 1.2, cloud.height * 1.3)

    for (const puff of cloud.puffs) {
      const puffX = x + puff.x * cloud.width
      const puffY = y + (puff.y - puff.ry * 0.3) * cloud.height
      const radius = Math.max(puff.rx * cloud.width, puff.ry * cloud.height) * 1.9
      const highlight = ctx.createRadialGradient(
        puffX - puff.rx * cloud.width * 0.22,
        puffY - puff.ry * cloud.height * 0.3,
        0,
        puffX,
        puffY,
        radius,
      )
      highlight.addColorStop(0, `rgba(255,255,255,${cloud.alpha * 0.72 * puff.alpha})`)
      highlight.addColorStop(0.42, `rgba(255,255,255,${cloud.alpha * 0.26 * puff.alpha})`)
      highlight.addColorStop(1, 'rgba(255,255,255,0)')
      ctx.fillStyle = highlight
      ctx.fillRect(puffX - radius, puffY - radius, radius * 2, radius * 2)
    }

    ctx.save()
    ctx.globalCompositeOperation = 'multiply'
    for (const puff of cloud.puffs) {
      const puffX = x + puff.x * cloud.width
      const puffY = y + (puff.y + puff.ry * 0.38) * cloud.height
      const radius = Math.max(puff.rx * cloud.width, puff.ry * cloud.height) * 1.45
      const puffShade = ctx.createRadialGradient(puffX, puffY, 0, puffX, puffY, radius)
      puffShade.addColorStop(0, `rgba(102, 139, 149, ${cloud.alpha * 0.22 * puff.alpha})`)
      puffShade.addColorStop(0.58, `rgba(145, 177, 181, ${cloud.alpha * 0.1 * puff.alpha})`)
      puffShade.addColorStop(1, 'rgba(180, 202, 202, 0)')
      ctx.fillStyle = puffShade
      ctx.fillRect(puffX - radius, puffY - radius * 0.7, radius * 2, radius * 1.4)
    }
    ctx.restore()

    const cloudShade = ctx.createLinearGradient(0, y, 0, y + cloud.height * 0.48)
    cloudShade.addColorStop(0, 'rgba(91, 133, 143, 0)')
    cloudShade.addColorStop(0.55, `rgba(86, 130, 140, ${cloud.alpha * 0.12})`)
    cloudShade.addColorStop(1, `rgba(70, 112, 124, ${cloud.alpha * 0.42})`)
    ctx.fillStyle = cloudShade
    ctx.fillRect(x - cloud.width * 0.6, y, cloud.width * 1.2, cloud.height * 0.55)
    ctx.restore()

    ctx.save()
    traceCloud(cloud, x, y - cloud.height * 0.025)
    ctx.strokeStyle = `rgba(255, 255, 255, ${cloud.alpha * 0.24})`
    ctx.lineWidth = Math.max(0.6, cloud.height * 0.009)
    ctx.lineJoin = 'round'
    ctx.stroke()
    ctx.restore()
  }

  function drawDaylight(t: number) {
    const sky = ctx.createLinearGradient(0, 0, 0, h)
    sky.addColorStop(0, '#96b8c6')
    sky.addColorStop(0.42, '#bfd7dc')
    sky.addColorStop(0.78, '#d3e0df')
    sky.addColorStop(1, '#c3d2d2')
    ctx.fillStyle = sky
    ctx.fillRect(0, 0, w, h)

    const sunX = w * 0.84 + (reduce ? 0 : Math.sin(t * 0.08) * w * 0.025)
    const sunY = h * 0.13 + (reduce ? 0 : Math.cos(t * 0.06) * h * 0.018)
    const sunRadius = Math.max(180, Math.min(w, h) * (0.44 + (reduce ? 0 : Math.sin(t * 0.34) * 0.018)))
    const halo = ctx.createRadialGradient(sunX, sunY, 0, sunX, sunY, sunRadius)
    halo.addColorStop(0, 'rgba(255, 248, 211, 0.48)')
    halo.addColorStop(0.28, 'rgba(255, 235, 174, 0.22)')
    halo.addColorStop(1, 'rgba(255, 235, 174, 0)')
    ctx.fillStyle = halo
    ctx.fillRect(sunX - sunRadius, sunY - sunRadius, sunRadius * 2, sunRadius * 2)

    ctx.save()
    ctx.globalCompositeOperation = 'screen'
    const beamShift = reduce ? 0 : Math.sin(t * 0.16) * w * 0.06
    const beam = ctx.createLinearGradient(sunX + beamShift, sunY, w * 0.34 + beamShift, h)
    beam.addColorStop(0, 'rgba(255, 245, 204, 0.14)')
    beam.addColorStop(1, 'rgba(255, 245, 204, 0)')
    ctx.fillStyle = beam
    ctx.beginPath()
    ctx.moveTo(sunX - 22 + beamShift, sunY + 8)
    ctx.lineTo(sunX + 48 + beamShift, sunY + 4)
    ctx.lineTo(w * 0.38 + beamShift, h)
    ctx.lineTo(w * 0.18 + beamShift, h)
    ctx.closePath()
    ctx.fill()
    ctx.restore()

    if (!reduce) {
      const sweepX = ((t * 0.14) % 1.6 - 0.3) * w
      const sweep = ctx.createLinearGradient(sweepX, 0, sweepX + w * 0.28, h)
      sweep.addColorStop(0, 'rgba(255, 255, 255, 0)')
      sweep.addColorStop(0.48, 'rgba(255, 250, 219, 0.08)')
      sweep.addColorStop(1, 'rgba(255, 255, 255, 0)')
      ctx.save()
      ctx.globalCompositeOperation = 'screen'
      ctx.fillStyle = sweep
      ctx.fillRect(0, 0, w, h)
      ctx.restore()
    }

    for (const cloud of clouds) {
      if (!reduce) cloud.x += cloud.speed
      if (cloud.x - cloud.width > w + 80) cloud.x = -cloud.width - 80
      const x = cloud.x + Math.sin(t * 0.05 + cloud.phase) * 12
      const y = cloud.y + Math.cos(t * 0.035 + cloud.phase) * 4
      drawCloud(cloud, x, y)
    }

    for (const particle of dust) {
      if (!reduce) {
        particle.x += 0.08
        particle.y -= 0.04
        if (particle.x > w + 4) particle.x = -4
        if (particle.y < -4) particle.y = h + 4
      }
      const twinkle = reduce ? 1 : 0.58 + Math.sin(t * 1.15 + particle.phase) * 0.42
      ctx.fillStyle = `rgba(255, 247, 211, ${particle.alpha * twinkle})`
      ctx.beginPath()
      ctx.arc(particle.x, particle.y, particle.radius, 0, Math.PI * 2)
      ctx.fill()
    }

    const lowerHaze = ctx.createLinearGradient(0, h * 0.55, 0, h)
    lowerHaze.addColorStop(0, 'rgba(255,255,255,0)')
    lowerHaze.addColorStop(1, 'rgba(74, 113, 122, 0.1)')
    ctx.fillStyle = lowerHaze
    ctx.fillRect(0, h * 0.55, w, h * 0.45)
  }

  function drawNight(t: number) {
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

    const aMul = cfg.nebulaAlpha
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
  }

  function frame(now: number) {
    if (!running) return
    const t = (now - t0) / 1000
    if (theme === 'light') drawDaylight(t)
    else drawNight(t)
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

let bootRaf = 0

function boot() {
  stop?.()
  if (bootRaf) cancelAnimationFrame(bootRaf)
  bootRaf = requestAnimationFrame(() => {
    bootRaf = requestAnimationFrame(() => {
      bootRaf = 0
      stop = startSky(props.intensity, props.theme)
    })
  })
}

onMounted(boot)
watch(() => [props.intensity, props.theme], boot)
onUnmounted(() => {
  if (bootRaf) cancelAnimationFrame(bootRaf)
  bootRaf = 0
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
