const PATTERNS = [
  { re: /\bdei\.gr\b|ppc\.gr|my[-.]?dei/i, name: 'ΔΕΗ' },
  { re: /\beydap\.gr\b/i, name: 'ΕΥΔΑΠ' },
  { re: /\beyath\.gr\b/i, name: 'ΕΥΑΘ' },
  { re: /\bcosmote\.gr\b|my\.cosmote/i, name: 'COSMOTE' },
  { re: /\bvodafone\.gr\b|vodafone\.com/i, name: 'Vodafone' },
  { re: /\bnova\.gr\b|wind\.gr/i, name: 'Nova' },
  { re: /\bote\.gr\b/i, name: 'OTE' },
  { re: /\bforthnet\.gr\b/i, name: 'Forthnet' },
  { re: /\bheron\.gr\b/i, name: 'ΗΡΩΝ' },
  { re: /\bprotergia\.gr\b/i, name: 'Protergia' },
  { re: /watt.and.volt|wattandvolt/i, name: 'Watt+Volt' },
  { re: /\bzenith\.gr\b/i, name: 'Zenith' },
  { re: /\bnetflix\.com\b/i, name: 'Netflix' },
  { re: /\bspotify\.com\b/i, name: 'Spotify' },
  { re: /disneyplus|disney\.com.*plus/i, name: 'Disney+' },
  { re: /\bamazon\.com\b.*(?:invoice|payment)|primevideo/i, name: 'Amazon Prime' },
  { re: /apple\.com.*(?:invoice|receipt)/i, name: 'Apple' },
  { re: /payments-noreply@google\.com|google.*(?:invoice|payment|τιμολόγ)/i, name: 'Google' },
]

export function detectProviderFromEmail(text: string): string | null {
  for (const { re, name } of PATTERNS) {
    if (re.test(text)) return name
  }
  return null
}
