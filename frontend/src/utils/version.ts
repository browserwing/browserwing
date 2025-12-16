// 当前版本号
export const CURRENT_VERSION = '0.0.1'

// 版本信息接口
export interface VersionInfo {
  version: string
  releaseDate: string
  features: string[]
  downloadUrl: string
}

// 从 GitHub 获取最新版本信息
export async function fetchLatestVersion(): Promise<VersionInfo | null> {
  try {
    const response = await fetch(
      'https://raw.githubusercontent.com/browserwing/browserwing/refs/heads/main/release-manifest/stable.json'
    )
    if (!response.ok) {
      throw new Error('Failed to fetch version info')
    }
    const data = await response.json()
    return data
  } catch (error) {
    console.error('Error fetching version info:', error)
    return null
  }
}

// 比较版本号 (返回 1 表示 v1 > v2, -1 表示 v1 < v2, 0 表示相等)
export function compareVersions(v1: string, v2: string): number {
  const parts1 = v1.split('.').map(Number)
  const parts2 = v2.split('.').map(Number)
  
  for (let i = 0; i < Math.max(parts1.length, parts2.length); i++) {
    const part1 = parts1[i] || 0
    const part2 = parts2[i] || 0
    
    if (part1 > part2) return 1
    if (part1 < part2) return -1
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
