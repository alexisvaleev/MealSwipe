/** Inline SVG icons — no icon dependency. 24×24 stroke icons, currentColor. */
import type { SVGProps } from 'react'

type P = SVGProps<SVGSVGElement>

function base(props: P): P {
  return {
    viewBox: '0 0 24 24',
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 2,
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
    'aria-hidden': true,
    ...props,
  }
}

export const IconHeart = (p: P) => (
  <svg {...base(p)}>
    <path d="M19.5 12.6 12 20l-7.5-7.4A5 5 0 1 1 12 6.3a5 5 0 1 1 7.5 6.3Z" />
  </svg>
)

export const IconHeartFilled = (p: P) => (
  <svg {...base({ ...p, fill: 'currentColor', stroke: 'none' })}>
    <path d="M19.5 12.6 12 20l-7.5-7.4A5 5 0 1 1 12 6.3a5 5 0 1 1 7.5 6.3Z" />
  </svg>
)

export const IconThumbUp = (p: P) => (
  <svg {...base(p)}>
    <path d="M6.6 10.2c.8 0 1.5-.4 2-1a9 9 0 0 1 2.9-2.5c-.7-.4-1.4-1-1.7-1.7a4.5 4.5 0 0 1-.3-1.7v-.6a.75.75 0 0 1 .75-.75 2.25 2.25 0 0 1 2.25 2.25c0 1.2-.3 2.3-.7 3.3-.3.5.1 1.2.7 1.2h3.1c1 0 2 .7 2.1 1.7l.1 1.3a12 12 0 0 1-2.7 7.5c-.4.5-1 .7-1.6.7h-1.2a4.5 4.5 0 0 1-1.4-.2L8.8 18.3a4.5 4.5 0 0 0-1.4-.2h-1.6m10.7-9.7h-2.2M5.8 18.4l.3.6c.2.4-.1.9-.5.9h-.9a2.1 2.1 0 0 1-2-1.4 12 12 0 0 1-.5-3.5c0-1.6.3-3 .8-4.4a2 2 0 0 1 2-1.3h1c.5 0 .8.6.5 1a9 9 0 0 0-1.3 4.7c0 1.2.2 2.3.6 3.4Z" />
  </svg>
)

export const IconThumbUpFilled = (p: P) => (
  <svg {...base({ ...p, fill: 'currentColor', stroke: 'none' })}>
    <path d="M7.5 18.5c-.4 0-.8-.2-1-.6a7.5 7.5 0 0 1-.5-2.8c0-1.7.6-3.3 1.6-4.6.2-.2.4-.3.6-.4.5-.2.8-.8 1-1.4a20.7 20.7 0 0 0 1.6-4.8C11 3.6 11.7 2.3 12.8 2.3c.5 0 1 .2 1.4.5.5.5.7 1.1.7 1.8 0 1.2-.3 2.5-.9 3.5-.2.3-.3.6-.3.9a.75.75 0 0 0 .75.75h5a1.5 1.5 0 0 1 1.5 1.2l.1.5c0 .4 0 .8-.1 1.3l-1 5.5c-.8 4.5-4.8 5.5-6.8 5.5-1 0-1.9-.2-2.7-.4l-.8-.1ZM3.75 7.5c1 0 1.75.8 1.75 1.75v10a1.75 1.75 0 0 1-3.5 0v-10c0-1 .8-1.75 1.75-1.75Z" />
  </svg>
)

export const IconX = (p: P) => (
  <svg {...base(p)}>
    <path d="M18 6 6 18M6 6l12 12" />
  </svg>
)

export const IconStar = (p: P) => (
  <svg {...base(p)}>
    <path d="m12 2.7 2.9 5.9 6.5.9-4.7 4.6 1.1 6.4L12 17.5l-5.8 3-1.1-6.4L.4 9.5l6.5-.9L12 2.7Z" />
  </svg>
)

export const IconStarFilled = (p: P) => (
  <svg {...base({ ...p, fill: 'currentColor', stroke: 'none' })}>
    <path d="m12 2.7 2.9 5.9 6.5.9-4.7 4.6 1.1 6.4L12 17.5l-5.8 3-1.1-6.4L.4 9.5l6.5-.9L12 2.7Z" />
  </svg>
)

export const IconUndo = (p: P) => (
  <svg {...base(p)}>
    <path d="M9 14 4 9l5-5" />
    <path d="M4 9h10.5a5.5 5.5 0 0 1 0 11H11" />
  </svg>
)

export const IconInfo = (p: P) => (
  <svg {...base(p)}>
    <circle cx="12" cy="12" r="9" />
    <path d="M12 16v-5M12 8h.01" />
  </svg>
)

export const IconLock = (p: P) => (
  <svg {...base(p)}>
    <rect x="4" y="11" width="16" height="10" rx="2" />
    <path d="M8 11V7a4 4 0 0 1 8 0v4" />
  </svg>
)

export const IconUnlock = (p: P) => (
  <svg {...base(p)}>
    <rect x="4" y="11" width="16" height="10" rx="2" />
    <path d="M8 11V7a4 4 0 0 1 7.7-1.5" />
  </svg>
)

export const IconSwap = (p: P) => (
  <svg {...base(p)}>
    <path d="M16 3l4 4-4 4" />
    <path d="M20 7H4M8 21l-4-4 4-4" />
    <path d="M4 17h16" />
  </svg>
)

export const IconTrash = (p: P) => (
  <svg {...base(p)}>
    <path d="M3 6h18M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
  </svg>
)

export const IconPlus = (p: P) => (
  <svg {...base(p)}>
    <path d="M12 5v14M5 12h14" />
  </svg>
)

export const IconMinus = (p: P) => (
  <svg {...base(p)}>
    <path d="M5 12h14" />
  </svg>
)

export const IconCart = (p: P) => (
  <svg {...base(p)}>
    <circle cx="9" cy="20" r="1.5" />
    <circle cx="18" cy="20" r="1.5" />
    <path d="M2 3h3l2.6 12.4a2 2 0 0 0 2 1.6h7.7a2 2 0 0 0 2-1.6L21.5 7H6" />
  </svg>
)

export const IconCalendar = (p: P) => (
  <svg {...base(p)}>
    <rect x="3" y="5" width="18" height="16" rx="2" />
    <path d="M8 3v4M16 3v4M3 10h18" />
  </svg>
)

export const IconGear = (p: P) => (
  <svg {...base(p)}>
    <circle cx="12" cy="12" r="3" />
    <path d="M19.4 15a1.7 1.7 0 0 0 .34 1.87l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.7 1.7 0 0 0-1.87-.34 1.7 1.7 0 0 0-1 1.55V21a2 2 0 1 1-4 0v-.09a1.7 1.7 0 0 0-1.11-1.55 1.7 1.7 0 0 0-1.87.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-1.55-1H3a2 2 0 1 1 0-4h.09A1.7 1.7 0 0 0 4.64 8.9a1.7 1.7 0 0 0-.34-1.87l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.7 1.7 0 0 0 1.87.34h.01a1.7 1.7 0 0 0 1-1.55V3a2 2 0 1 1 4 0v.09a1.7 1.7 0 0 0 1 1.55h.01a1.7 1.7 0 0 0 1.87-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.7 1.7 0 0 0-.34 1.87v.01a1.7 1.7 0 0 0 1.55 1H21a2 2 0 1 1 0 4h-.09a1.7 1.7 0 0 0-1.51 1Z" />
  </svg>
)

export const IconFlame = (p: P) => (
  <svg {...base(p)}>
    <path d="M12 22c4.4 0 7-2.8 7-6.5 0-3-2-5.5-3.5-7C14 7 13 5.5 13 3c-3 2-4.5 4.5-4.5 7-1-.5-1.7-1.3-2-2.5C5 9 5 11.5 5 13c0 5 2.6 9 7 9Z" />
  </svg>
)

export const IconClock = (p: P) => (
  <svg {...base(p)}>
    <circle cx="12" cy="12" r="9" />
    <path d="M12 7v5l3 2" />
  </svg>
)

export const IconCheck = (p: P) => (
  <svg {...base(p)}>
    <path d="m4 12.5 5 5L20 6.5" />
  </svg>
)

export const IconChevronLeft = (p: P) => (
  <svg {...base(p)}>
    <path d="m15 18-6-6 6-6" />
  </svg>
)

export const IconChevronRight = (p: P) => (
  <svg {...base(p)}>
    <path d="m9 18 6-6-6-6" />
  </svg>
)

export const IconCards = (p: P) => (
  <svg {...base(p)}>
    <rect x="7" y="7" width="13" height="15" rx="2.5" transform="rotate(6 7 7)" />
    <rect x="3" y="3" width="13" height="15" rx="2.5" transform="rotate(-6 3 3)" />
  </svg>
)

export const IconBowl = (p: P) => (
  <svg {...base(p)}>
    <path d="M4 12h16a8 8 0 0 1-16 0Z" />
    <path d="M9 8c0-2 1.5-2 1.5-4M13.5 8c0-2 1.5-2 1.5-4" />
  </svg>
)

export const IconSearch = (p: P) => (
  <svg {...base(p)}>
    <circle cx="11" cy="11" r="7" />
    <path d="m20 20-3.5-3.5" />
  </svg>
)

export const IconLogout = (p: P) => (
  <svg {...base(p)}>
    <path d="M9 21H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3" />
    <path d="m16 17 5-5-5-5M21 12H9" />
  </svg>
)

export const IconWallet = (p: P) => (
  <svg {...base(p)}>
    <path d="M20 7H4a2 2 0 0 1 0-4h14v4" />
    <path d="M4 7h16a1 1 0 0 1 1 1v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5" />
    <path d="M16 13.5h.01" />
  </svg>
)

export const IconUsers = (p: P) => (
  <svg {...base(p)}>
    <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
    <circle cx="9" cy="7" r="4" />
    <path d="M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" />
  </svg>
)

export const IconCopy = (p: P) => (
  <svg {...base(p)}>
    <rect x="9" y="9" width="13" height="13" rx="2" />
    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
  </svg>
)
