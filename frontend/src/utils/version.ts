// 当前版本号
export const CURRENT_VERSION = '1.0.0'

// 版本信息接口
export interface VersionInfo {
  version: string
  releaseDate: string
  features: string[]
  downloadUrl: string
  channel?: 'stable' | 'beta'
}

// 解析 semver 版本号，拆分为核心版本和预发布标签
// 例如: "1.2.3-beta.1" -> { core: [1, 2, 3], preRelease: ["beta", "1"] }
//       "1.2.3"         -> { core: [1, 2, 3], preRelease: null }
function parseSemver(version: string): { core: number[]; preRelease: string[] | null } {
  const [coreStr, ...preParts] = version.split('-')
  const core = coreStr.split('.').map(Number)
  const preRelease = preParts.length > 0
    ? preParts.join('-').split('.')
    : null
  return { core, preRelease }
}

const MANIFEST_BASE = 'https://raw.githubusercontent.com/browserwing/browserwing/refs/heads/main/release-manifest'

async function fetchManifest(channel: 'stable' | 'beta'): Promise<VersionInfo | null> {
  try {
    const response = await fetch(`${MANIFEST_BASE}/${channel}.json`)
    if (!response.ok) return null
    const data = await response.json()
    return { ...data, channel }
  } catch {
    return null
  }
}

// 从 GitHub 获取最新版本信息（同时检查 stable 和 beta 渠道）
export async function fetchLatestVersion(): Promise<VersionInfo | null> {
  try {
    const [stable, beta] = await Promise.all([
      fetchManifest('stable'),
      fetchManifest('beta'),
    ])

    if (stable && beta) {
      return compareVersions(beta.version, stable.version) > 0 ? beta : stable
    }
    return stable || beta
  } catch (error) {
    console.error('Error fetching version info:', error)
    return null
  }
}

// 比较版本号 (返回 1 表示 v1 > v2, -1 表示 v1 < v2, 0 表示相等)
// 支持 semver 预发布标签: 1.0.0-beta.1 < 1.0.0, 1.1.0-beta.1 > 1.0.0
export function compareVersions(v1: string, v2: string): number {
  const parsed1 = parseSemver(v1)
  const parsed2 = parseSemver(v2)

  // 先比较核心版本号 (major.minor.patch)
  for (let i = 0; i < Math.max(parsed1.core.length, parsed2.core.length); i++) {
    const part1 = parsed1.core[i] || 0
    const part2 = parsed2.core[i] || 0
    if (part1 > part2) return 1
    if (part1 < part2) return -1
  }

  // 核心版本相同，比较预发布标签
  // 按 semver 规范: 有预发布标签的版本 < 没有预发布标签的版本
  if (parsed1.preRelease && !parsed2.preRelease) return -1
  if (!parsed1.preRelease && parsed2.preRelease) return 1
  if (!parsed1.preRelease && !parsed2.preRelease) return 0

  // 两个都有预发布标签，逐段比较
  const pre1 = parsed1.preRelease!
  const pre2 = parsed2.preRelease!
  for (let i = 0; i < Math.max(pre1.length, pre2.length); i++) {
    if (i >= pre1.length) return -1
    if (i >= pre2.length) return 1

    const num1 = Number(pre1[i])
    const num2 = Number(pre2[i])
    const isNum1 = !isNaN(num1)
    const isNum2 = !isNaN(num2)

    if (isNum1 && isNum2) {
      if (num1 > num2) return 1
      if (num1 < num2) return -1
    } else if (isNum1 !== isNum2) {
      // 数字段 < 字符串段 (semver 规范)
      return isNum1 ? -1 : 1
    } else {
      // 都是字符串，按字典序比较
      if (pre1[i] > pre2[i]) return 1
      if (pre1[i] < pre2[i]) return -1
    }
  }

  return 0
}

// 检查是否有新版本
export function hasNewVersion(currentVersion: string, latestVersion: string): boolean {
  return compareVersions(latestVersion, currentVersion) > 0
}

// 本地存储的 key
const DISMISSED_VERSIONS_KEY = 'browserwing_dismissed_update_versions'

// 获取已关闭的版本列表
export function getDismissedVersions(): string[] {
  try {
    const stored = localStorage.getItem(DISMISSED_VERSIONS_KEY)
    return stored ? JSON.parse(stored) : []
  } catch {
    return []
  }
}

// 标记版本为已关闭
export function dismissVersion(version: string): void {
  const dismissed = getDismissedVersions()
  if (!dismissed.includes(version)) {
    dismissed.push(version)
    localStorage.setItem(DISMISSED_VERSIONS_KEY, JSON.stringify(dismissed))
  }
}

// 检查版本是否已被关闭
export function isVersionDismissed(version: string): boolean {
  return getDismissedVersions().includes(version)
}
