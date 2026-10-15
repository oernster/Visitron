// How deep the world map shades a country (FR-094): one of mapSteps steps of a
// single hue, rising on a logarithmic scale up to the largest count, since a
// few countries hold most of the visitors and equal steps would leave every
// other country in the palest one. Step 0 is a country of none, left as land.

/** The number of shaded steps; theme.css holds one colour for each. */
export const mapSteps = 5

/** The step count is shaded in when largest is the most any country had. */
export function shadeStep(count: number, largest: number): number {
  if (count <= 0 || largest <= 0) return 0
  return Math.min(mapSteps, Math.ceil((mapSteps * Math.log(count + 1)) / Math.log(largest + 1)))
}

/** One line of the map's key: the step and the counts it covers. */
export interface KeyStep {
  step: number
  low: number
  high: number
}

/**
 * The key beneath the map: each step that some count up to largest falls in,
 * with the range it covers. It walks the counts through shadeStep itself, so
 * the key cannot disagree with the shading.
 */
export function shadeKey(largest: number): KeyStep[] {
  const out: KeyStep[] = []
  for (let count = 1; count <= largest; count++) {
    const step = shadeStep(count, largest)
    const last = out[out.length - 1]
    if (last && last.step === step) last.high = count
    else out.push({ step, low: count, high: count })
  }
  return out
}
