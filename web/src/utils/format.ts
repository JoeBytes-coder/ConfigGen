/** 将后端 RFC3339 时间格式化为本地可读字符串 */
export function formatDateTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString('zh-CN', { hour12: false })
}

/** 拼接 image:tag 展示 */
export function imageLabel(image?: string, tag?: string): string {
  if (!image) return '-'
  return tag ? `${image}:${tag}` : image
}
