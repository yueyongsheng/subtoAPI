import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const layoutSource = readFileSync(resolve(dir, '../AuthLayout.vue'), 'utf8')
const styleSource = readFileSync(resolve(dir, '../../../style.css'), 'utf8')
const tailwindSource = readFileSync(resolve(dir, '../../../../tailwind.config.js'), 'utf8')

describe('AuthLayout branding', () => {
  it('uses the restrained authentication surface without decorative gradients or orbs', () => {
    expect(layoutSource).toContain('min-h-[100svh]')
    expect(layoutSource).toContain('data-testid="auth-card"')
    expect(layoutSource).not.toContain('Gradient Orbs')
    expect(layoutSource).not.toContain('bg-gradient-to-br')
    expect(layoutSource).not.toContain('text-gradient')
  })

  it('shows configured legal links only when login agreements are enabled', () => {
    expect(layoutSource).toContain('settings?.login_agreement_enabled !== true')
    expect(layoutSource).toContain("name: 'LegalDocument'")
    expect(layoutSource).toContain('legalDocuments.length > 0')
  })

  it('uses the logo blue as the primary action color and keeps teal as a secondary brand color', () => {
    expect(tailwindSource).toContain("600: '#126bd0'")
    expect(tailwindSource).toContain("brand: {")
    expect(styleSource).toContain('@apply bg-primary-600;')
    expect(styleSource).not.toContain('@apply bg-gradient-to-r from-primary-500 to-primary-600;')
  })
})
