const LANGUAGE_COLORS: Record<string, string> = {
  C: '#555555',
  'C#': '#178600',
  'C++': '#f34b7d',
  CSS: '#563d7c',
  Go: '#00add8',
  HTML: '#e34c26',
  Java: '#b07219',
  JavaScript: '#f1e05a',
  JSON: '#292929',
  Kotlin: '#a97bff',
  Markdown: '#083fa1',
  PHP: '#4f5d95',
  Python: '#3572a5',
  Ruby: '#701516',
  Rust: '#dea584',
  Shell: '#89e051',
  SQL: '#e38c00',
  Swift: '#f05138',
  TypeScript: '#3178c6',
  Vue: '#41b883',
  YAML: '#cb171e',
}

const FALLBACK_PALETTE = [
  '#3178c6',
  '#f1e05a',
  '#3572a5',
  '#00add8',
  '#dea584',
  '#b07219',
  '#e34c26',
  '#563d7c',
  '#89e051',
  '#c6538c',
  '#a97bff',
  '#701516',
]

function hashLanguageName(name: string): number {
  let h = 0
  for (let i = 0; i < name.length; i++) {
    h = (h * 31 + name.charCodeAt(i)) >>> 0
  }
  return h
}

export function languageColor(name: string): string {
  const known = LANGUAGE_COLORS[name]
  if (known) {
    return known
  }

  return FALLBACK_PALETTE[hashLanguageName(name) % FALLBACK_PALETTE.length]!
}
