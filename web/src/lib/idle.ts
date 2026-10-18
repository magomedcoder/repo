export function idleAge(iso?: string | null): string {
  if (!iso) {
    return ''
  }

  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) {
    return ''
  }

  const minutes = Math.max(0, (Date.now() - then) / 60000)
  if (minutes < 2) {
    return '1 min.'
  }
  
  if (minutes < 90) {
    return `${Math.round(minutes)} min.`
  }

  const hours = minutes / 60
  if (hours < 36) {
    return `${Math.round(hours)} hours`
  }

  const days = hours / 24
  if (days < 12) {
    return `${Math.round(days)} days`
  }

  const weeks = days / 7
  if (weeks < 8) {
    return `${Math.round(weeks)} weeks`
  }

  const months = days / 30
  if (months < 20) {
    return `${Math.round(months)} months`
  }

  return `${Math.round(days / 365)} years`
}
