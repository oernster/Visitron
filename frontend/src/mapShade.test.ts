import { describe, expect, it } from 'vitest'
import { mapSteps, shadeKey, shadeStep } from './mapShade'

describe('the map shading', () => {
  it('rises on a logarithmic scale from the fewest to the most', () => {
    expect(shadeStep(1, 100)).toBe(1)
    expect(shadeStep(10, 100)).toBe(3)
    expect(shadeStep(100, 100)).toBe(mapSteps)
    // Equal steps would put 10 of 100 in the first step with 1.
    expect(shadeStep(10, 100)).toBeGreaterThan(shadeStep(1, 100))
  })

  it('leaves a country of none unshaded', () => {
    expect(shadeStep(0, 100)).toBe(0)
    expect(shadeStep(0, 0)).toBe(0)
  })

  it('gives a lone count the deepest step', () => {
    expect(shadeStep(1, 1)).toBe(mapSteps)
  })

  it('keys each step with the counts it covers, agreeing with the shading', () => {
    const key = shadeKey(100)
    expect(key.map((k) => k.step)).toEqual([1, 2, 3, 4, 5])
    expect(key[0].low).toBe(1)
    expect(key[key.length - 1].high).toBe(100)
    for (const k of key) {
      expect(shadeStep(k.low, 100)).toBe(k.step)
      expect(shadeStep(k.high, 100)).toBe(k.step)
    }
    for (let i = 1; i < key.length; i++) expect(key[i].low).toBe(key[i - 1].high + 1)
  })

  it('keys only the steps a small largest reaches', () => {
    expect(shadeKey(1)).toEqual([{ step: mapSteps, low: 1, high: 1 }])
    expect(shadeKey(0)).toEqual([])
  })
})
